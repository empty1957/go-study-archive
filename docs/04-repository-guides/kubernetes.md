# Kubernetes の読み方

公式 repository: [kubernetes/kubernetes](https://github.com/kubernetes/kubernetes)

## 何を学ぶか

巨大な Go monorepo を分割して理解する方法、versioned API、declarative reconciliation、client/server の version skew、複数 SIG による ownership を学びます。

## 地図

repository は変化しますが、代表的な入口は次です。

- `cmd/kube-apiserver`: API server command の起動。
- `cmd/kube-controller-manager`: controller 群の process。
- `cmd/kube-scheduler`: scheduling process。
- `cmd/kubelet`: node agent。
- `pkg/`: Kubernetes repository 内の implementation。
- `staging/src/k8s.io/`: 別 module として公開される API、client、component library の staging tree。
- `test/`: integration / end-to-end / conformance 等の大規模 test assets。

正確な現在の tree と build 方法は公式 [repository README](https://github.com/kubernetes/kubernetes) と [community contributor guide](https://github.com/kubernetes/community/tree/master/contributors) を確認します。

## 読解 1: API request

題材: Pod の GET または作成。

1. versioned Pod type と REST path を確認。
2. API server の command/options から server construction を追う。
3. authentication → authorization → admission → validation → storage の順序を図にする。
4. etcd へ保存される型への conversion と defaulting を探す。
5. watch、resourceVersion、conflict が client にどう見えるか調べる。
6. 同じ機能の integration test と API conformance test を読む。

学習テーマは「HTTP handler の書き方」より、長期間進化する public API の境界です。

## 読解 2: controller

小さめの controller を 1 つ選びます。

```text
informer/watch -> local cache -> workqueue
                         ↓
                  sync/reconcile key
                         ↓
                 API update / requeue
```

確認する問い:

- cache が stale でも収束するのはなぜか。
- key が重複 enqueue されたらどうなるか。
- error と retry/backoff をどう扱うか。
- controller が所有する field / condition は何か。
- delete と finalizer にどんな race があるか。

## 読解 3: scheduler plugin

1 Pod の scheduling cycle を入口に、filter、score、bind の extension point を追います。plugin interface を増やすことの compatibility cost、snapshot/cache と API の observed state のずれ、parallelism と deterministic decision を観察します。

## 小さな contribution 候補

- docs と flag help の差を検証。
- validation の edge case に table-driven test を追加。
- context cancellation や error wrapping の改善候補を、既存慣習と照合。
- flaky test を再現し、原因と seed / log を報告。

Kubernetes は ownership が明確です。該当 SIG の contribution flow と issue 方針を確認してから作業します。

