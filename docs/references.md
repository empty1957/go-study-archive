# 参考資料

変化し得る仕様・要件は、blog の要約ではなく一次情報を確認します（確認日: 2026-08-17）。

## Go

- [A Tour of Go](https://go.dev/tour/): 対話的な言語入門。
- [Effective Go](https://go.dev/doc/effective_go): idiom の背景。新機能すべてを網羅する文書ではない点に注意。
- [Go Language Specification](https://go.dev/ref/spec): 最終的な言語仕様。
- [Go Memory Model](https://go.dev/ref/mem): goroutine 間で write が可視になる条件。
- [Go Modules Reference](https://go.dev/ref/mod): module、version、`go.mod` の仕様。
- [Go Blog: Working with Errors in Go 1.13](https://go.dev/blog/go1.13-errors): error wrapping と `errors.Is/As` の設計背景。
- [`errors` package documentation](https://pkg.go.dev/errors): error tree、`Is`、`As`、`Unwrap` の現在の契約。
- [Go Blog: Pipelines and cancellation](https://go.dev/blog/pipelines): pipeline と cancellation の基本。
- [`context` package documentation](https://pkg.go.dev/context): cancel の伝播、`CancelFunc`、deadline の現在の契約。
- [`net/http.Server` package documentation](https://pkg.go.dev/net/http#Server): `Serve`、`Shutdown`、`Close` の終了契約。
- [Data Race Detector](https://go.dev/doc/articles/race_detector): 実行方法、検出範囲、platform 要件。
- [Go Blog: Fuzzing](https://go.dev/doc/tutorial/fuzz): native fuzzing の入口。
- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments): review で頻出する idiom。

## Cloud native / CNCF

- [CNCF Projects](https://www.cncf.io/projects/): project と maturity の公式一覧。
- [CNCF Technical Oversight Committee](https://github.com/cncf/toc): project proposal / maturity process の最新情報を確認する入口。
- [Kubernetes documentation](https://kubernetes.io/docs/home/): architecture、API、運用。
- [Kubernetes: Pod Lifecycle](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/): termination flow、signal、grace period、EndpointSlice condition。
- [Kubernetes: Liveness, Readiness, and Startup Probes](https://kubernetes.io/docs/concepts/workloads/pods/probes/): 各 probe の目的、failure の効果、設定項目。
- [Kubernetes: Pods and Endpoints Termination Flow](https://kubernetes.io/docs/tutorials/services/pods-and-endpoint-termination-flow/): terminating endpoint を観測する公式演習。
- [Kubernetes: Container Lifecycle Hooks](https://kubernetes.io/docs/concepts/containers/container-lifecycle-hooks/): `preStop` と TERM の順序、grace period との関係。
- [Kubernetes: Deployments](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/): rolling update、availability、progress deadline。
- [Open Container Initiative Specs](https://github.com/opencontainers): image/runtime/distribution の仕様。
- [OpenTelemetry specification](https://opentelemetry.io/docs/specs/): telemetry API と data model。
- [Prometheus documentation](https://prometheus.io/docs/introduction/overview/): metrics と monitoring model。
- [SLSA specification](https://slsa.dev/spec/): software artifact supply-chain integrity。

## 読解対象の一次情報

- [Kubernetes repository](https://github.com/kubernetes/kubernetes)
- [containerd repository](https://github.com/containerd/containerd) と [plugin architecture](https://github.com/containerd/containerd/blob/main/docs/PLUGINS.md)
- [Prometheus repository](https://github.com/prometheus/prometheus)
- [etcd repository](https://github.com/etcd-io/etcd)
- [Raft paper](https://raft.github.io/raft.pdf)

## 読む順序

公式 tutorial → package docs → source/test → design proposal → specification の順で疑問を深めます。検索結果や生成された説明は入口にし、API 契約と security 判断は必ず対象 version の一次情報へ戻ります。
