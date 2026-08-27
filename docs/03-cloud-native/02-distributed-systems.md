# 分散システム

## 前提: timeout 後の結果は二値ではない

[HTTP service の timeout](../02-engineering/02-http-service.md#server-timeout)と [`context` の deadline](../02-engineering/01-concurrency-context.md)を先に実行してください。process をまたぐと、caller が観測する「失敗」と server が処理したかどうかは一致しません。

```text
request ──┬─ server 到達前に失敗 ───────────────> 未適用
          ├─ side effect 後に response を喪失 ──> 適用済みだが caller には不明
          └─ response を受信 ──────────────────> 結果を観測
```

timeout や connection reset は「未適用」の証明ではありません。retry は可用性機能であると同時に、同じ side effect を再要求する操作です。operation ID、idempotency key と durable deduplication、一意制約、または read-after-timeout で結果を照合できなければ、曖昧な失敗を自動再送しません。

HTTP method の名前だけにも依存しません。`PUT` や `DELETE` は HTTP semantics 上 idempotent ですが、application が同じ request で別の課金や通知を発生させれば end-to-end では安全ではありません。`POST` でも、key と payload の対応を durable に保存し、同じ key へ同じ結果を返す契約なら replay-safe にできます。

## retry を一つの判定契約にする

再試行ループを書く前に、次の順で判定します。どれか一つでも満たせなければ、待機せず caller へ結果を返します。

1. **分類**: dependency の契約に基づき permanent / transient / overload / ambiguous を分ける。
2. **再送安全性**: 同じ operation を適用済みでも replay-safe か証明する。
3. **所有層**: service mesh、SDK、application のうち一層だけが retry を所有する。
4. **負荷予算**: process / cluster で共有する retry budget から一回分を取得する。
5. **回数予算**: 初回を含む最大 attempt 数を超えない。
6. **時間予算**: backoff と次 attempt が request 全体の deadline に収まる。

### error 分類は status code の羅列ではない

| 観測 | 既定判断 | retry するために必要な追加契約 |
|---|---|---|
| validation、authentication、permission error | permanent | 入力・credential・policy の変更 |
| caller cancel / 全体 deadline 超過 | stop | 上位操作が新しい deadline でやり直す明示判断 |
| `429` / `503` と `Retry-After` | overload | replay-safe、共有 budget、残り時間 |
| dependency が transient と公開する `5xx` | transient | replay-safe と error code の versioned contract |
| timeout、EOF、connection reset | ambiguous | 適用済みでも安全、または未適用を照合可能 |
| conflict、precondition failure | domain decision | state を再読込して新しい操作として再評価 |

`429` や `503` だから無条件に再送するのではありません。`Retry-After` は待機の下限として扱い、その時間が全体 deadline を越えるなら即座に失敗させます。逆に server が明確な overload を返したのに即時 retry すると、回復のために空けた capacity を caller が埋め戻します。

### 多層 retry は掛け算になる

4 attempts を application、proxy、SDK の3層で行うと、最下流には最悪 `4 × 4 × 4 = 64` attempts が届きます。retry owner を一層にし、それ以外は deadline と分類済み error を伝播します。hedging は別 replica へ意図的に重複 request を送るため、同じ retry budget と replay-safe 条件に含めます。

## 時間と負荷を同時に予算化する

回数上限だけでは retry storm を止められません。平常 traffic が増えたとき、各 request が同じ回数を使えば追加負荷も比例して増えます。少なくとも次の三つを別々に持ちます。

- **total deadline**: 初回、全 backoff、全 attempt を含む利用者側の上限。
- **attempt limit**: 1 回の接続・処理・response に与える上限、意味のある処理に必要な最小時間、最大回数。
- **shared retry budget**: process / tenant / dependency 単位で一定 window に許す追加 attempt。通常 traffic 用 capacity と分離する。

shared budget は成功率が高い平常時には一時的 failure を隠し、dependency が広く失敗しているときは枯渇して fail fast します。公平性が必要なら tenant ごとに分け、priority の低い background retry が interactive traffic を奪わないようにします。budget が空でも local queue に無制限に積まず、元の error と `retry_budget_exhausted` を返します。

backoff の上限を `cap(n) = min(maxBackoff, base × 2^(n-1))` とすると、full jitter は `0 <= delay <= cap(n)` の一様な乱数です。同時に失敗した client の再送時刻を散らせます。ただし jitter は総量を制限しないため、shared budget の代替ではありません。server が有効な `Retry-After` を返した場合は `max(jitterDelay, retryAfter)` を使います。

## 実行例: retry planner

[`examples/retrycontract/retry.go`](../../examples/retrycontract/retry.go) は network call や `time.Sleep` を内包せず、証拠から `retry / stop`、delay、次 attempt timeout を返す純粋な planner です。

```go
decision, err := retrycontract.Decide(
    retrycontract.Policy{
        MaxAttempts: 4, TotalTimeout: 5 * time.Second,
        AttemptTimeout: time.Second,
        MinAttemptTime: 100 * time.Millisecond,
        BaseBackoff: 100 * time.Millisecond, MaxBackoff: 800 * time.Millisecond,
    },
    replaySafe,
    retrycontract.Failure{Kind: kind, RetryAfter: retryAfter},
    retrycontract.State{
        CompletedAttempts: completed,
        Elapsed: elapsed,
        RetryBudgetAvailable: budget.Available(),
    },
    randomFraction,
)
```

`replaySafe` は「たぶん大丈夫」という flag ではありません。idempotent な意図、durable な key/result、または未適用の証拠を operation contract から導出します。`Retry` の決定後、実行直前に budget token を原子的に取得し、取得競合に負けたら stop してください。planner と実行を分けることで、random、clock、network を使わず境界条件を test できます。

```console
go test ./examples/retrycontract
go test -count=100 ./examples/retrycontract
```

対応する [table test](../../examples/retrycontract/retry_test.go) は response loss、budget 枯渇、attempt 上限、`Retry-After` が deadline 外にある場合、最後の attempt timeout の切り詰めを固定します。

### failure injection 演習

1. 同じ idempotency key の create を受けた fake server が、一度だけ保存後に connection を閉じるようにする。
2. retry 後も作成された resource ID が一つで、payload を変えた同じ key は conflict になることを確認する。
3. 100 client を同時に失敗させ、jitter なし / full jitter の attempt 開始時刻を histogram にする。
4. dependency を失敗させ続け、shared budget 枯渇後に追加 call が止まり、元の request latency も total deadline 内に収まることを確認する。
5. application と proxy の両方で retry を有効にした失敗例を作り、最下流の attempt 数を数えてから owner を一層へ戻す。

## 観測で retry を原因として見えるようにする

[可観測性と SRE](03-observability-sre.md#retry-を観測する)では、初回 traffic と追加 attempt を分けます。少なくとも operation と dependency ごとに次を関連付けます。

- logical operation 数、全 attempt 数、`attempts / operations` の増幅率。
- failure kind、retry / stop reason、retry budget の利用・枯渇。
- backoff 秒数、`Retry-After` 採用数、total / attempt deadline 超過。
- idempotency dedup hit、key conflict、重複 side effect。

request ID、idempotency key、raw error は高 cardinality なので log / trace に置き、metric label は bounded な `operation`、`dependency`、`failure_kind`、`decision` に限定します。retry 成功率だけを見ると「利用者には成功したが下流を10倍呼んだ」劣化を見逃します。

## 一貫性を要件から選ぶ

すべてを strong consistency にすると latency と availability の代償があり、すべて eventual にすると利用者が異常な状態を見る可能性があります。invariant ごとに決めます。

- job の二重課金防止: 一意制約や fencing を使う強い保証。
- dashboard の集計: 数秒遅れてもよい eventual consistency。
- configuration rollout: version を持つ snapshot と段階適用。

CAP は「3 つから常に 2 つ」だけで覚えず、network partition 中に consistency と availability のどちらを選ぶかという議論に使います。平常時の latency や運用性も別の trade-off です。

## queue と delivery

exactly-once は system 全体の end-to-end invariant として考えます。broker が exactly-once を謳っても、DB side effect や外部 API まで自動的に一度にはなりません。多くの場合、at-least-once delivery + idempotent consumer + durable deduplication が実用的です。queue の redelivery と HTTP retry が同じ side effect に到達するなら、別々の回数ではなく一つの operation ID と budget で追跡します。

## Raft の最低限

Raft は replicated log による consensus algorithm です。leader が log entry を提案し、多数派への replication 後に commit します。term、leader election、log matching、commit index、state machine application を区別します。

Raft library を使っても、snapshot、storage durability、membership change、transport、backpressure、read semantics、運用は application の責任です。[etcd の読解ガイド](../04-repository-guides/etcd.md) で実装を追います。

## failure test matrix

| 注入 | 確認すること |
|---|---|
| response loss | retry で二重 side effect がなく、同じ結果を回収できる |
| dependency overload | `Retry-After`、jitter、shared budget が追加負荷を制限する |
| latency / timeout | total deadline と attempt timeout を分け、無意味な最終 attempt を作らない |
| retry owner の重複 | 最下流の attempt 数が層ごとの積にならない |
| process kill | durable state から再開できる |
| network partition | 一貫性選択どおりに振る舞う |
| disk full/corruption | 安全に停止し診断可能 |
| clock skew | lease や TTL が危険な ownership を作らない |
| version skew | rolling upgrade の compatibility |
