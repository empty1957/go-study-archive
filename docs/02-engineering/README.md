# 02: Go エンジニアリング

このセクションでは、動く Go コードを「停止できる、失敗を説明できる、変更を検証できる service」へ進めます。[Go の基礎](../01-foundations/README.md)にある値の所有権・小さな interface・error chain が前提です。

## 読む順序と一本の問い

| 順序 | 章 | 判断できるようになること | 教材内の証拠 |
|---|---|---|---|
| 1 | [並行処理と context](01-concurrency-context.md) | goroutine の owner、cancel、join、error path を決める | [`examples/pipeline`](../../examples/pipeline)、[`cmd/taskapi`](../../cmd/taskapi) |
| 2 | [HTTP service](02-http-service.md) | HTTP と domain の境界、timeout、drain を設計する | [`internal/task/http.go`](../../internal/task/http.go) |
| 3 | [テスト戦略](03-testing.md) | 非決定的な停止を event で再現する | lifecycle / readiness tests |
| 4 | [設計と依存関係](04-architecture.md) | dependency と failure mode を変更理由で分ける | `task.Service` / `task.Store` |
| 5 | [性能と profiling](05-performance.md) | concurrency を増やす前に saturation を測る | bounded pipeline |

全章を貫く問いは「この仕事を始めた owner は、停止を通知した後、完了と error をどこで回収するか」です。`context.CancelFunc` を呼ぶだけでは仕事の完了は証明できません。

## 実行してから読む

```console
go test ./examples/pipeline ./cmd/taskapi ./internal/task
go test -race ./...
go run ./cmd/taskapi
```

1. [`TestMapCancellationClosesOutput`](../../examples/pipeline/pipeline_test.go) で callback が cancel を観測し、全 worker の join 後に output が閉じる順序を追う。
2. [`TestServeHTTPDrainsThenJoinsServer`](../../cmd/taskapi/main_test.go) で readiness、shutdown、join の順序を追う。
3. server の shutdown timeout を短くし、graceful shutdown 失敗時に何を失うか説明する。
4. [技能チェックリスト](../checklist.md#並行処理)へ commit や実験ログを証拠として残す。

