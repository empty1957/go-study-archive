# コンテナと Kubernetes

この章では「Pod を動かせる」ではなく、更新・縮退・削除の途中でも request の結果を説明できることを目標にします。先に [HTTP service の health endpoint](../02-engineering/02-http-service.md#health-endpoint) と [停止契約](../02-engineering/01-concurrency-context.md#実行例-2-http-server-を-drain-して-join-する)を読み、`go test ./cmd/taskapi` が通る状態から始めてください。

## container の境界

container image は root filesystem と実行設定の配布形式で、実行時の隔離は主に OS の namespace、cgroup、capability 等が担います。container を VM と同じ security boundary だと無条件に仮定しません。

Go binary を image にする際の判断項目は次のとおりです。

- multi-stage build と小さな runtime imageを使い、更新対象と攻撃面を減らす。
- non-root user、read-only root filesystem、不要 capability の削除を既定にする。
- architecture / OS と CGO の有無を明示し、実行環境との差をなくす。
- digest pinning、SBOM、署名、provenance で source と artifact を結ぶ。
- signal を受け取る process が PID 1 になることを確認する。shell wrapper を挟むなら signal を転送する。
- writable path、CA certificate、timezone data が本当に必要かを実行時に検証する。

image を小さくすることと debug 可能性には trade-off があります。shell のない image を採用するなら、ephemeral container、profile、structured log など別の診断経路を先に用意します。

## control loop と application の契約

利用者は desired state を API Server に宣言し、controller が observed state を望ましい状態へ収束させます。scheduler は未配置 Pod の node を選び、kubelet は node 上で Pod lifecycle を実現します。永続 state の中核には etcd が使われます。

```text
kubectl/client -> API Server -> etcd
                       ↑
             controllers / scheduler
                       ↓
                    kubelet -> container runtime -> PID 1
```

control loop は process 内の transaction や goroutine を知りません。Kubernetes は replacement を作れますが、application を自動的に reliable にはしません。application は少なくとも次を契約として持ちます。

- signal を処理し、有限の猶予内で新規受付停止、in-flight work の完了、resource close、join を行う。
- readiness を起動中・一時的な処理不能・明示的な drain に対応させ、liveness と混同しない。
- resource request / limit を計測に基づき設定し、overload 時は無制限に queue しない。
- replica 間で local disk / memory state を共有できる、または Pod IP や起動順序が永続 identity になると仮定しない。
- disruption、reschedule、response loss、duplicate execution に耐える。side effect は [冪等性と retry](02-distributed-systems.md#retry-の設計)まで含めて設計する。

## Pod 削除で並行して起きること

`kubectl delete pod` や rollout で削除が始まると、単一の直列処理ではなく次が並行して進みます。

```text
API Server: deletionTimestamp と grace period を記録
       ├─ control plane: EndpointSlice を terminating / ready=false へ
       └─ kubelet: preStop（あれば）→ PID 1 へ TERM → 猶予切れで KILL

Go process: signal受信 → ready=false → routing待機 → Shutdown → join → exit
```

重要な因果関係は次のとおりです。

1. `terminationGracePeriodSeconds` は `preStop` と process の終了を合わせた Pod 全体の予算です。`preStop` の後で TERM が送られるため、hook が長いほど application の残り時間は減ります。
2. terminating endpoint は EndpointSlice から即座に消えるとは限りませんが、通常 traffic 用の `ready` condition は `false` になります。削除だけを目的に readiness probe を追加する必要はありません。
3. control plane、proxy、外部 load balancer、client の connection pool への反映には時間差があります。`ready=false` は「この瞬間から request が一件も来ない」という同期 barrier ではありません。
4. 猶予切れの `SIGKILL` では cleanup できません。強制終了を例外扱いせず、途中で切れても壊れない durable write と冪等性が必要です。

EndpointSlice の `serving` condition を使う drain-aware client もありますが、すべての load balancer が同じ挙動だと仮定しません。実際の経路で観測してください。

## probe は失敗原因ごとに分ける

| probe | 答える質問 | failure の効果 | 典型的な失敗 |
|---|---|---|---|
| startup | 初期化は完了したか | 規定回数失敗で restart | warm-up 中に liveness が先に殺す |
| readiness | 今この Pod へ traffic を送れるか | Service の通常 backend から外す | 全依存を含めて小障害で全 replica を外す |
| liveness | restart しなければ回復不能か | container を restart | overload を deadlock と誤認し再起動を連鎖させる |

probe endpoint は小さく、bounded で、side effect なしにします。本教材の [`/healthz` と `/readyz`](../../internal/task/http.go) は意図的に分離されています。readiness に downstream 全体を直列問い合わせすると、依存障害時に残っている処理能力まで失う場合があります。「その依存なしでは正しい response を一件も返せないか」を基準に含める対象を決めます。

## 終了時間を予算化する

希望ではなく上限の和で決めます。

```text
Pod grace period
  > preStop 上限
  + routing propagation 待機
  + application shutdown 上限
  + sidecar / runtime / scheduler の安全余白
```

Task API の例では Pod に 30 秒を与え、application の総 shutdown budget を 20 秒、routing propagation 待機を 5 秒にすると、`http.Server.Shutdown` が使える残りは最大約 15 秒です。残り 10 秒は process exit と platform 側の余白にします。値は例であり、request latency 分布、keep-alive、外部 LB の反映時間、sidecar の有無を実測して変えます。

```yaml
spec:
  terminationGracePeriodSeconds: 30
  containers:
    - name: taskapi
      image: your-registry.example/taskapi@sha256:REPLACE_ME
      env:
        - name: TASKAPI_ROUTING_DRAIN_DELAY
          value: 5s
      ports:
        - name: http
          containerPort: 8080
      readinessProbe:
        httpGet: {path: /readyz, port: http}
        periodSeconds: 2
        failureThreshold: 1
      livenessProbe:
        httpGet: {path: /healthz, port: http}
        periodSeconds: 10
        failureThreshold: 3
```

[`cmd/taskapi`](../../cmd/taskapi/main.go) は `TASKAPI_ROUTING_DRAIN_DELAY` を Go duration として読みます。signal 後に readiness を落とし、その待機を application の 20 秒の総予算に含めてから `Shutdown` します。負数や不正な値では起動に失敗します。待機を deadline の外へ置くと、in-flight request 用の時間を無意識に使い切るためです。

`preStop: sleep` と application の待機を理由なく重ねないでください。前者は TERM より前、後者は TERM より後に動き、どちらも同じ Pod grace period を消費します。外部 LB の仕様上 `preStop` が必要な場合は、合計時間を再計算します。

## 実行可能な演習

まず cluster を使わず、順序と deadline を deterministic な test で確認します。

```console
go test -run 'TestServeHTTP(DrainsThenJoinsServer|ForceCloses)' -v ./cmd/taskapi
```

次に routing 待機を有効にして server を起動し、別 terminal から `/readyz` と request を繰り返しながら `Ctrl+C` で停止します。

```console
TASKAPI_ROUTING_DRAIN_DELAY=5s go run ./cmd/taskapi
```

PowerShell では次のように設定します。

```powershell
$env:TASKAPI_ROUTING_DRAIN_DELAY = '5s'
go run ./cmd/taskapi
```

cluster では rollout 中に次を同時に観測し、時刻を記録します。

```console
kubectl get pods -w
kubectl get endpointslices -l kubernetes.io/service-name=taskapi -o yaml -w
kubectl logs -f deployment/taskapi
kubectl rollout status deployment/taskapi --timeout=2m
```

合格条件は「Pod が消えた」ではありません。少なくとも、terminating endpoint の condition、`shutdown requested` と routing 待機の log、in-flight request の結果、強制終了の有無を一つの timeline にし、`terminationGracePeriodSeconds` を 3 秒など意図的に短くした失敗実験との差を説明してください。観測項目は [終了を user impact へ結ぶ](03-observability-sre.md#termination-を観測する)へ続きます。

## operator を作る前に

CRD/controller は declarative domain に有効ですが、versioned API と長期保守の責任を生みます。Helm chart や通常の service API で十分か、reconciliation が user value を生むか、削除/finalizer/upgrade を安全に扱えるかを先に検討します。rollout と compatibility は [API、互換性、リリース](05-api-release.md)で扱います。
