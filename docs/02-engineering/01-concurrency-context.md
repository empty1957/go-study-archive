# 並行処理と context

前提: [値と alias](../01-foundations/02-language-core.md#値渡しと共有しないは同じではない)、[小さな interface と error chain](../01-foundations/03-interfaces-errors.md)を先に説明できるようにします。並行処理では memory だけでなく、処理の寿命と error の所有権も共有されるからです。

## cancel は完了ではない

`CancelFunc` は「止まってほしい」という signal を broadcast します。対象 goroutine の終了を待つ機能ではありません。安全な lifecycle は次の因果関係を持ちます。

```text
owner が開始
  -> 子へ context を渡す
  -> cancel / deadline で停止を通知
  -> 子が cleanup して return
  -> owner が join
  -> error と終了状態を公開
```

goroutine を始める前に lifecycle ledger を書きます。

| 問い | 決めること |
|---|---|
| owner | 誰が goroutine を開始し、寿命に責任を持つか |
| completion | channel close、`Wait`、結果受信のどれが完了を証明するか |
| cancellation | caller cancel、deadline、内部 error のどれで停止するか |
| join | 誰が全 goroutine の終了を待つか |
| error path | 最初の error、複数 error、正常終了をどこへ返すか |

この表を埋められない goroutine は、leak、停止漏れ、失われた error の候補です。

## channel と mutex は所有権で選ぶ

- ownership を移動する work queue / stream: channel が自然。
- 共有する小さな state と複数 field の invariant: mutex が単純。
- 1 回だけ初期化: `sync.Once`。
- goroutine 集合の join: `sync.WaitGroup`。cancel と error 回収は別途必要。
- 単一の counter / flag: `sync/atomic`。複数 field をまとめて守る用途にはしない。

「channel なら安全」ではありません。channel の外の memory を並行更新すれば data race になります。一方、mutex で data race を消しても「重複実行してはいけない」という論理的な race condition は残り得ます。

## channel close は protocol の一部

原則として sender が close します。receiver は「今後もう送られない」と判断できないからです。close は完了 event の broadcast で、queue の値を捨てる操作ではありません。閉じた channel への send と二重 close は panic します。

```go
func generate(ctx context.Context, values []int) <-chan int {
    out := make(chan int)
    go func() {
        defer close(out) // producer が唯一の sender / closer
        for _, v := range values {
            select {
            case out <- v:
            case <-ctx.Done():
                return
            }
        }
    }()
    return out
}
```

send 側が `ctx.Done()` を選ばないと、consumer が受信をやめた後に永遠に block できます。consumer も途中で受信をやめるなら cancel しなければなりません。これは双方の protocol です。

## context が伝えるもの、伝えないもの

- `context.Context` は第 1 引数にし、struct に保存しない。
- request lifetime を越える処理に request context を流用しない。
- `nil` を渡さず、意図に応じて `Background` または `TODO` を使う。
- cancel function は作った側が全 control-flow path で呼ぶ。
- value は API 境界を通る request-scoped metadata に限定し、設定や dependency injection に使わない。
- `context.Canceled` と `context.DeadlineExceeded` を metric や retry 判断で区別する。

context は協調的です。関数が blocking I/O や channel operation で `ctx.Done()` を観測しなければ、cancel しても止まりません。さらに、止まったことを証明するには別の join が必要です。

## 実行例 1: bounded pipeline を所有権で読む

[`examples/pipeline/pipeline.go`](../../examples/pipeline/pipeline.go) の `Map` は worker 数を上限にし、callback に context を渡します。

```go
results := pipeline.Map(ctx, 4, inputs,
    func(ctx context.Context, input Input) Output {
        return callDownstream(ctx, input)
    },
)
for result := range results {
    consume(result)
}
```

内部の ownership は次のとおりです。

```text
caller --cancel--> producer + workers
producer --------close(jobs)--------> workers
workers ---------WaitGroup.Done-----> join goroutine
join goroutine ---close(results)-----> caller の range 終了
```

- producer だけが `jobs` を close する。
- 各 worker は callback から return して `Done` を呼ぶ。
- join goroutine だけが、全 worker 終了後に `results` を close する。
- caller は結果を途中で捨てるなら、受信をやめる前に cancel する。

callback が context を無視して永久に block すれば、`Map` だけでは強制停止できません。subprocess、database driver、HTTP client など境界ごとの cancel 方法と timeout が必要です。

## bounded concurrency と backpressure

入力ごとに無制限に goroutine を作ると、memory、socket、下流 DB の connection を使い切ります。worker 数または semaphore で同時実行数を制限します。queue の容量は throughput の魔法ではなく、burst を一時吸収する待ち場所です。

queue が満杯のときは product requirement に基づき選びます。

| 方針 | 利点 | 代償 |
|---|---|---|
| block | loss を避け、自然に backpressure を返す | upstream の latency が伸びる |
| reject | overload を明示し resource を守る | caller に retry / UX 設計が必要 |
| drop | stale telemetry などを新鮮に保てる | loss を契約・metric にする必要 |
| buffer を増やす | 短い burst を吸収する | failure を遅らせ、memory と待ち時間を増やす |

## 実行例 2: HTTP server を drain して join する

[`cmd/taskapi/main.go`](../../cmd/taskapi/main.go) は process signal を受けた後、次の順序で停止します。

1. readiness を false にして、新しい traffic の配送対象から外す。
2. canceled な signal context とは別に、shutdown 用 deadline を作る。
3. `http.Server.Shutdown` で listener を閉じ、idle connection を閉じ、active request を待つ。
4. deadline を超えたら `Close` で強制終了し、timeout error を保持する。
5. `ListenAndServe` の結果 channel を受信し、server goroutine を join する。

`Shutdown` に signal context を渡すと、それはすでに canceled なので drain できません。だから shutdown 用には新しい `context.Background()` から有限 deadline を作ります。一方、deadline を無限にすると終了できない request が deploy 全体を止めます。timeout は「何秒なら利用者処理を守り、いつ強制終了へ切り替えるか」という運用上の判断です。

`ListenAndServe` は正常な `Shutdown` 後にも `http.ErrServerClosed` を返します。これは期待した終了として正規化し、bind failure などそれ以外の error は owner が回収します。

readiness と liveness の役割は [HTTP service の health endpoint](02-http-service.md#health-endpoint)を参照してください。

## よくある失敗とトレードオフ

| 失敗 | 観測される症状 | 修正の方向 |
|---|---|---|
| goroutine を開始して結果を受け取らない | error 消失、test 終了後も処理が残る | owner が result channel / `Wait` を必ず join |
| cancel だけ呼び、完了を待たない | resource close と処理が競合する | cancel → join → resource close の順序を定義 |
| consumer が無言で range を抜ける | producer が send で block | consumer が cancel する protocol を API に明記 |
| callback が context を無視する | pipeline の output が閉じない | context-aware I/O と deadline を末端まで伝播 |
| readiness と liveness を同時に落とす | drain 中に process が再起動される | traffic 制御と process 生存判定を分離 |
| shutdown timeout だけ返して join しない | server goroutine が残り得る | `Close` fallback 後に serve result を回収 |

## sleep せず lifecycle をテストする

並行テストで「100 ms 待てば終わるはず」と仮定すると、遅い環境では flaky になります。[`main_test.go`](../../cmd/taskapi/main_test.go) と [`pipeline_test.go`](../../examples/pipeline/pipeline_test.go) は channel を観測点として使います。

- callback に入った event を待ってから cancel する。
- shutdown が呼ばれた event を待って、まだ join 前なら関数が返らないことを確認する。
- worker / server の終了を許可し、結果 channel が閉じることを有限 timeout で確認する。
- race detector は実行された経路だけを検査するため、cancel 前後の経路を明示的に通す。

詳細は [テスト戦略: lifecycle test](03-testing.md#lifecycle-test-は順序を固定する)へ続きます。

## 演習

1. `TestMapCancellationClosesOutput` から callback の `<-ctx.Done()` を外し、なぜ test が終了しないか lifecycle ledger で説明する。
2. `Map` の結果を 1 件だけ受け取って cancel しない consumer を作り、どの send が block するか stack dump で確認する。
3. Task API に 1 秒 block する handler を追加し、shutdown timeout の前後で response と process error がどう変わるか test する。
4. shutdown の `DeadlineExceeded` と caller cancel を別 metric に数える設計を書く。
5. shared map をわざと無同期で更新し、`go test -race` の作成 goroutine と競合 stack を読む。

