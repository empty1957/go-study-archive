# 04: 実在リポジトリのコードリーディング

対象はすべて Go を中核に持つ、CNCF ecosystem の代表的プロジェクトです。main branch は変化するため、読む際は release tag または commit SHA を記録してください。

| Repository | 主に学ぶこと | 最初のガイド |
|---|---|---|
| [Kubernetes](https://github.com/kubernetes/kubernetes) | 巨大 codebase、宣言的 API、controller、compatibility | [kubernetes.md](kubernetes.md) |
| [containerd](https://github.com/containerd/containerd) | daemon、gRPC、plugin、runtime boundary | [containerd.md](containerd.md) |
| [Prometheus](https://github.com/prometheus/prometheus) | ingestion、query、TSDB、operability | [prometheus.md](prometheus.md) |
| [etcd](https://github.com/etcd-io/etcd) | consensus、WAL、state machine、robustness | [etcd.md](etcd.md) |

## 1 repository を読む手順

1. README、go.mod、CONTRIBUTING、release/support policy を読む。
2. release tag を checkout し、build/test の最小 command を確認する。
3. `cmd/` の `main` から dependency の組み立てを追う。
4. public API / protobuf から 1 operation を選ぶ。
5. handler → domain/service → storage/runtime の順に call graph を手書きする。
6. その operation 名で test、metric、log、docs を検索する。
7. timeout、cancel、retry、partial failure、version skew の扱いを記録する。
8. `git blame` ではなく、関連 PR / design doc を読み「なぜ」を確認する。
9. 小さな test を変更して仮説を検証する。

## 読解ノートテンプレート

```markdown
# Repository / tag / commit
## User-visible operation
## Entry point and call path
## Important interfaces and ownership
## State and invariants
## Cancellation / retry / failure behavior
## Tests and observability
## Compatibility constraints
## One design I would reuse
## One trade-off I would choose differently
## Upstream question or contribution idea
```

## contribution の入口

最初から core algorithm を変更しません。再現可能な bug report、docs と実装の差、error message、test coverage の穴は価値ある入口です。issue の assignment / label / contributor guide を尊重し、変更前に scope を合意します。

