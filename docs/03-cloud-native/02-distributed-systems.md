# 分散システム

## 失敗は部分的で曖昧

timeout は「相手が処理しなかった」ことを意味しません。処理後に response だけ失われた可能性があります。したがって retry の前に operation ID、idempotency key、deduplication、read-after-timeout の戦略が必要です。

## consistency を要件から選ぶ

すべてを strong consistency にすると latency と availability の代償があり、すべて eventual にすると利用者が異常な状態を見る可能性があります。invariant ごとに決めます。

例:

- job の二重課金防止: 一意制約や fencing を使う強い保証。
- dashboard の集計: 数秒遅れてもよい eventual consistency。
- configuration rollout: version を持つ snapshot と段階適用。

CAP は「3 つから常に 2 つ」だけで覚えず、network partition 中に consistency と availability のどちらを選ぶかという議論に使います。平常時の latency や運用性も別の trade-off です。

## Raft の最低限

Raft は replicated log による consensus algorithm です。leader が log entry を提案し、多数派への replication 後に commit します。term、leader election、log matching、commit index、state machine application を区別します。

Raft library を使っても、snapshot、storage durability、membership change、transport、backpressure、read semantics、運用は application の責任です。[etcd の読解ガイド](../04-repository-guides/etcd.md) で実装を追います。

## retry の設計

- retry 可能な error を分類する。
- exponential backoff に jitter を入れる。
- request 全体の deadline と attempt timeout を分ける。
- retry 回数だけでなく budget を設ける。
- server overload (`429/503`) を尊重する。
- retry storm を観測する。
- non-idempotent operation を無条件に retry しない。

## queue と delivery

exactly-once は system 全体の end-to-end invariant として考えます。broker が exactly-once を謳っても、DB side effect や外部 API まで自動的に一度にはなりません。多くの場合、at-least-once delivery + idempotent consumer + durable deduplication が実用的です。

## failure test matrix

| 注入 | 確認すること |
|---|---|
| response loss | retry で二重 side effect がない |
| latency / timeout | deadline と backpressure が働く |
| process kill | durable state から再開できる |
| network partition | 一貫性選択どおりに振る舞う |
| disk full/corruption | 安全に停止し診断可能 |
| clock skew | lease や TTL が危険な ownership を作らない |
| version skew | rolling upgrade の compatibility |

