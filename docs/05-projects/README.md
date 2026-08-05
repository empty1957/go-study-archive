# 05: 作ることで学ぶ

## Project A: Task CLI（Phase 1）

JSON file に task を保存する CLI。追加、一覧、完了、削除、import/export を実装します。

重点: package、I/O、error、atomic file replacement、table test、CLI contract。

## Project B: Task service（Phase 2）

この repository の Task API を発展させます。

順序:

1. pagination と stable sort。
2. SQL repository と migration。
3. optimistic concurrency / version field。
4. authentication / per-project authorization。
5. request ID、metrics、trace。
6. rate / concurrency limit。
7. OpenAPI と compatibility test。

## Project C: Durable worker（Phase 3）

API が job を enqueue し、worker が lease を取得して処理します。process kill、重複 delivery、timeout を test します。

重点: idempotency key、transactional outbox、retry budget、dead letter、backpressure、graceful drain。

## Project D: Declarative controller（Phase 4）

desired state を API に保存し、controller が外部 resource を reconcile します。

重点: work queue、rate-limited retry、finalizer、condition、leader election、versioned API。

## Project E: [Capstone](capstone.md)

狭い cloud native problem を解く production-grade control plane を設計し、公開 OSS として運営します。

## 各 project の Definition of Done

- README に problem / non-goals / architecture / runbook がある。
- clean machine で build/test できる。
- error / cancellation / shutdown / resource limit の test がある。
- logs/metrics/traces から主要 failure を診断できる。
- security assumptions と threat model がある。
- upgrade / rollback / backup restore を試した記録がある。
- 他者が docs だけで利用し、issue を報告できる。

