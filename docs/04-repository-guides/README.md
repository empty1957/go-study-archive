# 04: 実在リポジトリのコードリーディング

対象はすべて Go を中核に持つ、CNCF ecosystem の代表的プロジェクトです。main branch は変化するため、読む際は release tag または commit SHA を記録してください。

| Repository | 主に学ぶこと | 最初のガイド |
|---|---|---|
| [Kubernetes](https://github.com/kubernetes/kubernetes) | 巨大 codebase、宣言的 API、controller、compatibility | [kubernetes.md](kubernetes.md) |
| [containerd](https://github.com/containerd/containerd) | daemon、gRPC、plugin、runtime boundary | [containerd.md](containerd.md) |
| [Prometheus](https://github.com/prometheus/prometheus) | ingestion、query、TSDB、operability | [prometheus.md](prometheus.md) |
| [etcd](https://github.com/etcd-io/etcd) | consensus、WAL、state machine、robustness | [etcd.md](etcd.md) |

## このセクションの進め方

最初の題材は [Kubernetes の Pod 削除と EndpointSlice](kubernetes.md)です。1つの user-visible な状態変化を、API owner、controller、node agent、pure function、test、実測へ分解する型を身につけてから、containerd、Prometheus、etcd へ横展開します。

読む量ではなく、次の evidence chain を完成させることを目標にします。

```text
利用者から見える invariant
  -> version を固定した source
  -> owner と state transition
  -> unit / controller test
  -> failure injection
  -> live observation
```

## 1 repository を読む手順

1. README、go.mod、CONTRIBUTING、release/support policy を読む。
2. release tag を checkout し、build/test の最小 command を確認する。
3. `cmd/` の `main` から dependency の組み立てを追う。
4. public API / protobuf から user-visible な invariant を 1 つ選ぶ。
5. call graph だけでなく、process を越える state、watch、queue、ownership を図にする。
6. 境界を越える値の名前で source、test、metric、log、docs を検索する。
7. unit test、controller/integration test、live observation がそれぞれ何を証明するか分ける。
8. timeout、cancel、retry、partial failure、stale cache、version skew の扱いを記録する。
9. `git blame` だけで理由を推測せず、関連 PR / design doc と互換性制約を確認する。
10. 小さな test または縮小モデルを変更し、仮説を反証する。

## 読解ノートテンプレート

```markdown
# Repository / tag / commit
## User-visible operation
## Entry point and call path
## Important interfaces and ownership
## State and invariants
## Owners, async boundaries, and ordering assumptions
## Cancellation / retry / failure behavior
## Evidence ladder: unit / integration / live
## Compatibility constraints
## Hypothesis disproved by a test
## One design I would reuse
## One trade-off I would choose differently
## Upstream question or contribution idea
```

## contribution の入口

最初から core algorithm を変更しません。再現可能な bug report、docs と実装の差、error message、test coverage の穴は価値ある入口です。issue の assignment / label / contributor guide を尊重し、変更前に scope を合意します。
