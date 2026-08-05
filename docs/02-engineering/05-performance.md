# 性能と profiling

## 観測してから最適化する

性能作業の順序:

1. user-facing SLO と workload を定義。
2. 同じ入力・machine・version で baseline を保存。
3. CPU / allocation / blocking / mutex / trace から bottleneck を特定。
4. 1 つ変更し、correctness test と benchmark を再実行。
5. production 相当負荷で tail latency と resource を確認。

平均 latency だけでなく p50 / p95 / p99、throughput、error、CPU、memory、GC、queue depth を一緒に見ます。

## Go の道具

```console
go test -bench=. -benchmem ./...
go test -run='^$' -bench=BenchmarkX -count=10 ./path
go test -cpuprofile=cpu.out -bench=BenchmarkX ./path
go tool pprof cpu.out
go test -memprofile=mem.out -bench=BenchmarkX ./path
go test -trace=trace.out ./path
go tool trace trace.out
```

service では `net/http/pprof` を管理 network に限定して公開します。public internet にそのまま出すと内部情報漏洩と負荷の危険があります。

## よくある誤最適化

- allocation を 1 個減らして code を読めなくするが、I/O が支配的。
- goroutine を増やして下流 DB の queue latency を悪化させる。
- object pool で retained memory と複雑性を増やす。
- map iteration や scheduler の偶然の順序に依存する。
- benchmark 中に compiler が結果を除去する。
- laptop の 1 回の数値だけで結論を出す。

## capacity の考え方

Little's Law の直感 `concurrency ≈ throughput × latency` は、同時実行数・queue の見積もりに役立ちます。ただし burst、分布、retry、downstream limit がある実システムでは負荷試験で検証します。saturation 前に admission control と backpressure が働く設計にします。

