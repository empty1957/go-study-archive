# 並行処理と context

## concurrency は構造を先に決める

goroutine を始める前に 4 点を答えます。

1. 誰が開始するか。
2. 何を完了とするか。
3. 誰がキャンセルするか。
4. 誰が終了を待ち、error を回収するか。

答えがない goroutine は leak、停止漏れ、失われた error の候補です。

## channel と mutex の選択

- ownership を移動する work queue / stream: channel が自然。
- 共有する小さな state を保護: mutex が単純。
- 1 回だけ初期化: `sync.Once`。
- goroutine 集合の終了を待つ: `sync.WaitGroup`（error/cancel も必要なら別途設計）。
- 原子的 counter: `sync/atomic`。複数 field の invariant には mutex。

「channel なら安全」ではありません。channel の外にある memory を並行更新すれば race になります。

## channel ownership

原則として sender が close します。receiver は「もう送られない」と判断できないからです。close は終了 event の broadcast で、値を捨てる操作ではありません。閉じた channel への send と二重 close は panic します。

```go
func generate(ctx context.Context, values []int) <-chan int {
    out := make(chan int)
    go func() {
        defer close(out)
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

send 側も `ctx.Done()` を選ばないと、consumer 終了後に永遠に block し得ます。

## context の規約

- request lifetime を越える処理に request context を流用しない。
- `context.Context` は第 1 引数にし、struct に保存しない。
- `nil` を渡さず、必要なら `context.Background()` を使う。
- cancel function は作った側が必ず呼ぶ。
- value は request-scoped metadata のうち API 境界を通るものだけ。設定や dependency injection に使わない。
- `context.Canceled` と `context.DeadlineExceeded` を運用上区別する。

## bounded concurrency

入力ごとに無制限に goroutine を作ると、memory、socket、下流 DB を使い切ります。worker 数または semaphore で上限を決めます。queue が満杯なら block、drop、reject のどれにするかを product requirement として選びます。

## graceful shutdown

shutdown の典型順序:

1. 新規 traffic を readiness から外す。
2. accept を止める。
3. deadline 内で in-flight request を待つ。
4. worker へ cancel を伝える。
5. queue / buffer を flush する。
6. resource を close し、終了 status を決める。

## 演習

- `examples/pipeline` の consumer を途中で停止しても goroutine が終了することを test する。
- worker 数を 1 / 4 / 20 に変えて throughput と memory を計測する。
- context timeout と caller cancel を別々の metric に数える設計を書く。
- shared map をわざと無同期で更新し、`go test -race` の出力を読む。

