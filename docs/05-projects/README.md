# 05: 作ることで学ぶ

このセクションでは機能数ではなく、**仮説を小さく実装し、失敗を注入し、証拠から次の操作を判断できること**を完成度とします。各 project は前段の成果物を捨てず、境界と運用契約を一つずつ増やします。

## Project のつながり

| Project | 追加する境界 | 主な失敗 | 次へ進む証拠 |
|---|---|---|---|
| A: Task CLI（Phase 1） | process ↔ file | 中断書き込み、壊れた入力 | atomic replacement と error test |
| B: Task service（Phase 2） | client ↔ HTTP ↔ store | timeout、競合更新、過負荷 | API contract と cancellation test |
| C: Durable worker（Phase 3） | API ↔ queue ↔ worker | 重複配送、process kill、lease 失効 | recovery と idempotency test |
| D: Declarative controller（Phase 4） | desired ↔ observed state | 部分障害、stale owner、非収束 | invariant と failure injection |
| E: [Capstone](capstone.md) | release ↔ 利用者・運用者 | version skew、誤判定、復旧失敗 | 段階 rollout と rollback drill |

章の知識へ戻るときは、[package と API 境界](../01-foundations/04-packages-api.md)、[HTTP service](../02-engineering/02-http-service.md)、[分散システム](../03-cloud-native/02-distributed-systems.md)、[API と release](../03-cloud-native/05-api-release.md) の順に、いま追加している境界だけを読み直します。

## Project A: Task CLI（Phase 1）

JSON file に task を保存する CLI。追加、一覧、完了、削除、import/export を実装します。まず「一時 file を flush してから rename する」失敗経路を test し、書き込み途中の process kill で既存 data を失わないことを示します。

判断課題: file lock を導入する前に、単一 writer という制約で十分か。複数 process 対応の複雑さと利用者の実需要を比較します。

## Project B: Task service（Phase 2）

この repository の Task API を次の順序で発展させます。

1. pagination と stable sort。
2. SQL repository と migration。
3. optimistic concurrency / version field。
4. authentication / per-project authorization。
5. request ID、metrics、trace。
6. rate / concurrency limit。
7. OpenAPI と compatibility test。

一度に足さず、各段階で「旧 client が新 server を使えるか」「timeout 後に retry して安全か」「上限超過を明示的に拒否するか」を test にします。

## Project C: Durable worker（Phase 3）

API が job を enqueue し、worker が lease を取得して処理します。process kill、response loss、重複 delivery、timeout を注入します。

`exactly-once` を看板にする代わりに、at-least-once delivery と idempotent side effect、または fencing token で守る範囲を明記します。retry 回数だけでなく、通常 traffic を圧迫しない retry budget と queue 上限も測ります。

## Project D: Declarative controller（Phase 4）

desired state を API に保存し、controller が外部 resource を reconcile します。work queue、rate-limited retry、finalizer、condition、leader election、versioned API を、収束に必要な順で追加します。

判断課題: leader election が本当に必要か、複数 reconciler が同じ操作をしても安全な設計にできないか。選出は可用性の仕組みであって、side effect の冪等性や fencing の代替ではありません。

## Project E: [Capstone](capstone.md)

狭い cloud native problem を解く production-grade control plane を設計し、公開 OSS として運営します。最初の production rollout の前に、下記の release evidence packet を作ります。

## Release evidence packet

PR や release ごとに、同じ artifact identity と観測 window に結び付いた次の証拠を保存します。

```markdown
## Release candidate
- commit / image digest:
- 変更する invariant:
- backward-compatible な schema/API の範囲:

## Gate
- 観測 window / 最小 sample 数:
- promote 閾値:
- 即時 rollback 条件:
- hold（証拠不足）条件:

## Recovery
- rollback command と所要時間:
- forward-only migration の復旧手順:
- backup/restore drill の記録:

## Result
- Promote / Hold / Rollback:
- 判断時刻と観測 link:
- 判断者と残る risk:
```

`Hold` は失敗ではありません。sample が少ないのに成功とみなす誤判定を防ぐ、fail-closed な結果です。逆に invariant 違反は sample 数を待たず `Rollback` します。binary を戻しても data migration や外部 side effect は戻らないため、「rollback 可能」は事前に演習した具体的な復旧経路を意味します。

[実行可能な rollout gate](../../examples/rolloutgate/gate.go) は、この順序を小さな pure function にしています。

```console
go test -v ./examples/rolloutgate
```

演習:

1. [table test](../../examples/rolloutgate/gate_test.go) の `MinRequests` を下げ、少数 sample の誤判定がどの test で見えるか説明する。
2. error rate は改善したが latency が悪化する観測を追加し、一つの平均値だけでは判定できないことを示す。
3. backward-incompatible migration を仮定し、`Rollback` を roll forward / traffic stop / restore のどれに置き換えるか decision record に残す。

## 各 project の Definition of Done

- README に problem / non-goals / architecture / runbook がある。
- clean machine で build/test できる。
- error / cancellation / shutdown / resource limit の test がある。
- logs/metrics/traces から主要 failure を診断できる。
- security assumptions と threat model がある。
- release evidence packet に artifact、閾値、判断、観測 link がそろっている。
- upgrade / rollback / backup restore を試し、所要時間と失敗を記録した。
- 他者が docs だけで利用し、issue を報告できる。
