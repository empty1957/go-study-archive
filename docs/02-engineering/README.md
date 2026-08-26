# 02: Go エンジニアリング

このセクションでは、動く Go コードを「停止できる、失敗を説明できる、変更を検証できる service」へ進めます。[Go の基礎](../01-foundations/README.md)にある値の所有権・小さな interface・error chain が前提です。

## 読む順序と一本の問い

| 順序 | 章 | 判断できるようになること | 教材内の証拠 |
|---|---|---|---|
| 1 | [並行処理と context](01-concurrency-context.md) | goroutine の owner、cancel、join、error path を決める | [`examples/pipeline`](../../examples/pipeline)、[`cmd/taskapi`](../../cmd/taskapi) |
| 2 | [HTTP service](02-http-service.md) | body と同時処理の resource 予算、明示的な overload 拒否を設計する | [`internal/task/http.go`](../../internal/task/http.go) |
| 3 | [テスト戦略](03-testing.md) | 非決定的な停止と resource 境界を event で再現する | lifecycle / admission tests |
| 4 | [設計と依存関係](04-architecture.md) | dependency と failure mode を変更理由で分ける | `task.Service` / `task.Store` |
| 5 | [性能と profiling](05-performance.md) | concurrency を増やす前に saturation を測る | bounded pipeline |

全章を貫く問いは「この仕事を開始してよいかを誰が決め、開始した owner は停止通知後の完了と error をどこで回収するか」です。`context.CancelFunc` を呼ぶだけでは仕事の完了は証明できず、capacity を超える仕事を無制限に開始すれば停止可能な設計にもなりません。

## 実行してから読む

```console
go test ./examples/pipeline ./cmd/taskapi ./internal/task
go test -race ./...
go run ./cmd/taskapi
```

1. [`TestMapCancellationClosesOutput`](../../examples/pipeline/pipeline_test.go) で callback が cancel を観測し、全 worker の join 後に output が閉じる順序を追う。
2. [`TestHandlerRejectsOverloadBeforeStartingWork`](../../internal/task/http_test.go) で capacity 超過を仕事の開始前に拒否する順序を追う。
3. [`TestServeHTTPDrainsThenJoinsServer`](../../cmd/taskapi/main_test.go) で readiness、shutdown、join の順序を追う。
4. `TASKAPI_MAX_IN_FLIGHT` と shutdown timeout を変え、過負荷時と終了時に何を失うか説明する。
5. [技能チェックリスト](../checklist.md#service--production)へ commit、負荷条件、実験ログを証拠として残す。

