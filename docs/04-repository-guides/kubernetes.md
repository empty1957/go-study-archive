# Kubernetes の読み方

公式 repository: [kubernetes/kubernetes](https://github.com/kubernetes/kubernetes)

このガイドは「Pod を削除すると EndpointSlice がどう変わるか」を題材に、巨大な Go repository を user-visible な invariant から横断して読む方法を扱います。運用上の終了時間予算は [コンテナと Kubernetes](../03-cloud-native/01-containers-kubernetes.md)、`context` と graceful shutdown は [並行処理と停止契約](../02-engineering/01-concurrency-context.md)を先に読んでください。

## 今回固定する版

main branch の行番号や実装は変わるため、以下は Kubernetes [`v1.36.2`](https://github.com/kubernetes/kubernetes/tree/v1.36.2)、commit [`24e2b02af5543d7910c2bb074c7264df5a8f0467`](https://github.com/kubernetes/kubernetes/commit/24e2b02af5543d7910c2bb074c7264df5a8f0467) を基準にします（確認日: 2026-08-18）。別版を読む場合は tag と commit SHA を読解ノートに残し、リンクの `v1.36.2` を置き換えてください。

```console
git clone --depth=1 --branch v1.36.2 https://github.com/kubernetes/kubernetes.git
cd kubernetes
git rev-parse HEAD
```

期待する SHA は `24e2b02af5543d7910c2bb074c7264df5a8f0467` です。release tag と手元の source が一致してから読み始めます。

## 先に invariant を書く

Pod に `deletionTimestamp` が設定され、Pod の `Ready` condition がまだ `True` なら、通常の Service では対応する endpoint は次の状態になります。

```text
ready=false, serving=true, terminating=true
```

これは「container はまだ request を処理できるが、通常 traffic の新規送信先からは外す」という互換性を保った表現です。ただし `Service.spec.publishNotReadyAddresses=true` は `ready` を `true` に上書きします。flag 名だけから「未準備 Pod だけに効く」と推測せず、実装と test で例外を確認します。

この invariant を次の因果に分解します。

```text
DELETE Pod
   |
   v
API server が grace period と deletionTimestamp を永続化
   | informer/watch                         | kubelet が Pod 更新を観測
   v                                        v
EndpointSlice controller                container runtime
Pod 更新を service key へ投影            preStop -> StopContainer(grace)
   |
   v
desired endpoint を再計算
ready = publishNotReady || (serving && !terminating)
```

左右の枝は同じ Pod 状態から進む非同期処理で、source 上の直列 call chain ではありません。API の DELETE 応答、EndpointSlice 更新、`preStop`、process への signal の間に同期 barrier があると読まないことが重要です。

## source map: owner と変換点を追う

| 問い | owner / source | 読む値と判断 |
|---|---|---|
| 削除猶予と timestamp はどこで決まるか | API server の [`BeforeDelete`](https://github.com/kubernetes/kubernetes/blob/v1.36.2/staging/src/k8s.io/apiserver/pkg/registry/rest/delete.go#L75-L174) と Pod 固有の [`CheckGracefulDelete`](https://github.com/kubernetes/kubernetes/blob/v1.36.2/pkg/registry/core/pod/strategy.go#L162-L194) | 二段階削除、request の値、Pod spec、未配置・終了済み Pod の例外 |
| Pod 更新がなぜ reconcile を起動するか | EndpointSlice controller の [event handler](https://github.com/kubernetes/kubernetes/blob/v1.36.2/pkg/controller/endpointslice/endpointslice_controller.go#L122-L136) と [`onPodUpdate`](https://github.com/kubernetes/kubernetes/blob/v1.36.2/pkg/controller/endpointslice/endpointslice_controller.go#L528-L534) | add/update/delete を `PodProjectionKey` へ変換 |
| どの変更を意味のある差とみなすか | [`GetPodUpdateProjectionKey`](https://github.com/kubernetes/kubernetes/blob/v1.36.2/staging/src/k8s.io/endpointslice/util/controller_utils.go#L50-L107) | resourceVersion だけの変化を捨て、labels・IP・readiness・deletion を分類 |
| どの Service を再計算するか | [`syncPod`](https://github.com/kubernetes/kubernetes/blob/v1.36.2/pkg/controller/endpointslice/endpointslice_controller.go#L453-L503) | Pod labels と Service selector から service key を queue へ入れる |
| desired endpoints はどう作られるか | [`reconcileByAddressType`](https://github.com/kubernetes/kubernetes/blob/v1.36.2/staging/src/k8s.io/endpointslice/reconciler.go#L163-L232) | informer cache の Pod 群を port map ごとに再計算 |
| condition の真理値はどこか | [`podToEndpoint`](https://github.com/kubernetes/kubernetes/blob/v1.36.2/staging/src/k8s.io/endpointslice/utils.go#L37-L50) | `Ready`、`DeletionTimestamp`、`PublishNotReadyAddresses` から 3 condition を導出 |
| container 側の grace はどう消費されるか | kubelet の [`killContainer`](https://github.com/kubernetes/kubernetes/blob/v1.36.2/pkg/kubelet/kuberuntime/kuberuntime_container.go#L858-L925) | `preStop` の経過を差し引き、CRI `StopContainer` へ残り時間を渡す |

「削除」を一語検索して全体を追うより、境界を越える値を一つずつ追います。この題材では `GracePeriodSeconds`、`DeletionTimestamp`、`PodReady`、`EndpointConditions` が観測可能な接続点です。

## controller の非同期性を読む

EndpointSlice controller は informer cache と workqueue を使います。Pod 更新は即座に API write を呼ぶのではなく、概ね次の段階を通ります。

1. informer が old/new Pod を event handler へ渡す。
2. semantic diff が endpoint membership に影響する変更だけを残す。
3. Pod labels に合う Service key を queue へ追加する。
4. worker が cache 上の Service、Pod、EndpointSlice を読み直す。
5. reconciler が desired set と existing set の差を create/update/delete する。
6. write failure は rate limit 付きで retry され、stale cache は明示的な error になる。

ここから分かる trade-off は、event ごとの差分をそのまま書く単純さより、最新 snapshot から収束させる冪等性と batching を選んでいることです。一方で API write の直後に EndpointSlice が変わる保証はなく、cache の遅延、queue の待ち、retry を観測に含める必要があります。

## 証拠の範囲を広げながら test を読む

同じ挙動でも test layer が答える問いは異なります。

1. [`TestCheckGracefulDelete`](https://github.com/kubernetes/kubernetes/blob/v1.36.2/pkg/registry/core/pod/strategy_test.go#L369-L448) で削除猶予の例外を確認する。
2. [`Test_podChanged`](https://github.com/kubernetes/kubernetes/blob/v1.36.2/staging/src/k8s.io/endpointslice/util/controller_utils_test.go#L448-L608) の `mark for deletion` case で `deletionTimestamp` が reconcile 対象になることを確認する。
3. [`TestPodToEndpoint`](https://github.com/kubernetes/kubernetes/blob/v1.36.2/staging/src/k8s.io/endpointslice/utils_test.go#L221-L472) で condition の真理値表と `publishNotReadyAddresses` の例外を確認する。
4. controller test の [`Terminating pods`](https://github.com/kubernetes/kubernetes/blob/v1.36.2/pkg/controller/endpointslice/endpointslice_controller_test.go#L792-L905) で、Pod 群から実際に作られる EndpointSlice を確認する。
5. kubelet の [`TestKillContainerGracePeriod`](https://github.com/kubernetes/kubernetes/blob/v1.36.2/pkg/kubelet/kuberuntime/kuberuntime_container_test.go#L914-L1000) で runtime に渡す grace の優先順位を確認する。
6. 最後に公式の [Pod と endpoint の termination flow](https://kubernetes.io/docs/tutorials/services/pods-and-endpoint-termination-flow/)を cluster で再現し、source の因果と時系列の観測を対応させる。

unit test が通っても watch delivery や API write の遅延は証明できません。逆に cluster 実験だけでは、`ready=false` になった分岐条件や例外を網羅できません。pure function → controller → component → live cluster と証拠を積み上げます。

## この repository で動かす縮小モデル

Kubernetes module を依存に追加せず、3 condition の導出だけを [`examples/endpointsliceconditions`](../../examples/endpointsliceconditions/conditions.go) に写した学習用モデルがあります。

```console
go test -v ./examples/endpointsliceconditions
```

test は `PodReady`、deletion、`publishNotReadyAddresses` の全 8 組み合わせを検証します。次の順で改造すると、互換性の理由を test failure として読めます。

1. `publishNotReadyAddresses` の項を式から外し、どの case が壊れるか予想する。
2. `Ready` を `Serving` と同じ値にし、terminating endpoint の backward compatibility が壊れることを確認する。
3. condition を計算する前後に時間待ちを追加しない。pure な state projection と、非同期 propagation の遅延を別責務として保つ。

このモデルは Kubernetes API の代替実装ではありません。source から抽出した invariant を安価に反証するための executable note です。

## upstream source で行う演習

clone した `v1.36.2` で、まず狭い test を実行します。初回は Kubernetes の依存 build に時間と disk を使います。

```console
go test ./staging/src/k8s.io/endpointslice -run TestPodToEndpoint -count=1
go test ./staging/src/k8s.io/endpointslice/util -run Test_podChanged -count=1
go test ./pkg/controller/endpointslice -run TestSyncService -count=1
go test ./pkg/registry/core/pod -run TestCheckGracefulDelete -count=1
```

読解ノートには最低限、次を残します。

- `publishNotReadyAddresses=false/true` の真理値表。
- API server、EndpointSlice controller、kubelet の owner と、各 branch が失敗した場合の user-visible state。
- `deletionTimestamp` 設定から EndpointSlice write までに待ち得る場所。
- unit test では証明できず、cluster 実験が必要な仮説。
- code と docs が食い違った場合、対象版と根拠 link を添えた upstream question。

## よくある読み違い

- **直列化する**: diagram の上下を実行順序だと決めつける。別 process・別 watch の枝には明示的な同期がない。
- **関数名だけで結論を出す**: `publishNotReadyAddresses` のような例外は boolean expression と table-driven test まで読む。
- **unit test を end-to-end guarantee に広げる**: cache、queue、API write、network propagation は別の evidence が必要。
- **main の URL を記録する**: 後日同じ行が別実装を指す。tag と commit SHA を固定する。
- **staging mirror へ直接 contribution する**: `k8s.io/endpointslice` の公開 repository は staging から同期されるため、変更元と contributor guide を確認する。

## 次の読解

この手順を使えるようになったら、同じ endpoint state を消費する proxy / load balancer 側へ進みます。その際も「Kubernetes が condition を書いた」と「すべての data plane が反映した」を分け、実装・test・実測の三つをそろえてください。
