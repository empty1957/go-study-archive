# 参考資料

変化し得る仕様・要件は、blog の要約ではなく一次情報を確認します（確認日: 2026-08-06）。

## Go

- [A Tour of Go](https://go.dev/tour/): 対話的な言語入門。
- [Effective Go](https://go.dev/doc/effective_go): idiom の背景。新機能すべてを網羅する文書ではない点に注意。
- [Go Language Specification](https://go.dev/ref/spec): 最終的な言語仕様。
- [Go Memory Model](https://go.dev/ref/mem): goroutine 間で write が可視になる条件。
- [Go Modules Reference](https://go.dev/ref/mod): module、version、`go.mod` の仕様。
- [Go Blog: Working with Errors in Go 1.13](https://go.dev/blog/go1.13-errors): error wrapping と `errors.Is/As` の設計背景。
- [`errors` package documentation](https://pkg.go.dev/errors): error tree、`Is`、`As`、`Unwrap` の現在の契約。
- [Go Blog: Pipelines and cancellation](https://go.dev/blog/pipelines): pipeline と cancellation の基本。
- [Go Blog: Fuzzing](https://go.dev/doc/tutorial/fuzz): native fuzzing の入口。
- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments): review で頻出する idiom。

## Cloud native / CNCF

- [CNCF Projects](https://www.cncf.io/projects/): project と maturity の公式一覧。
- [CNCF Technical Oversight Committee](https://github.com/cncf/toc): project proposal / maturity process の最新情報を確認する入口。
- [Kubernetes documentation](https://kubernetes.io/docs/home/): architecture、API、運用。
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
