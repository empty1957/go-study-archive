# Capstone: 分散ジョブ・コントロールプレーン

これは実装テーマの例です。先に user interview を行い、実在する未解決問題が別なら置き換えてください。

## problem statement

Kubernetes 内外の worker に長時間 job を安全に配布し、重複実行・worker 消失・段階 rollout を扱いながら、利用者が 1 つの宣言的 API から状態と失敗理由を理解できるようにする。

## non-goals（初期）

- 汎用 workflow DSL。
- 任意 code の sandbox 実行。
- global multi-region strong consistency。
- billing system。
- Kubernetes scheduler の置換。

## architecture sketch

```text
CLI / API clients
       │
       ▼
API server ─────> durable store
       │              ▲
       ▼              │ status / lease
reconcile scheduler ──┘
       │
       ▼
bounded queue / stream ──> workers ──> external side effects
          │                  │
          └── metrics/traces/logs ──> observability
```

## milestone 0: single process

- in-memory desired/current state。
- 1 reconcile loop、fake worker。
- deterministic clock と failure injection。
- invariant: 1 active lease per job generation。

## milestone 1: durable single node

- PostgreSQL 等への durable state。
- transaction と idempotency key。
- crash recovery、migration、backup/restore。
- HTTP/gRPC API と version policy。

## milestone 2: multiple API/controller nodes

- leader election または partitioned ownership。
- fencing token で stale worker の result を拒否。
- at-least-once event と idempotent apply。
- queue saturation / admission control。

## milestone 3: production operations

- SLO と burn-rate alert。
- OpenTelemetry trace、bounded-cardinality metrics、structured log。
- load / soak / chaos / version-skew test。
- canary、upgrade、rollback、disaster recovery drill。

## milestone 4: ecosystem

- stable plugin / worker protocol（本当に複数 implementation が出てから）。
- conformance suite と compatibility matrix。
- adopter docs、integration examples、security audit。
- governance と複数組織 maintainer。

## 必須 invariant

1. 完了済み job の generation は再び active にならない。
2. stale lease holder の結果は fencing token で拒否される。
3. client timeout 後の retry で job を重複作成しない。
4. status は desired generation のどこまで観測したかを示す。
5. queue 上限を越える workload は memory を増やし続けず、明示的に拒否・遅延される。

invariant ごとに unit、integration、failure test を最低 1 つ持たせます。

## 最初の API 案

```json
{
  "apiVersion": "jobs.example.io/v1alpha1",
  "kind": "Job",
  "metadata": {"name": "report-2026-08", "generation": 1},
  "spec": {
    "operation": "generate-report",
    "timeoutSeconds": 900,
    "maxAttempts": 3,
    "idempotencyKey": "report-2026-08"
  },
  "status": {
    "observedGeneration": 1,
    "phase": "Pending",
    "conditions": []
  }
}
```

schema を実装前に固定せず、3 つの client use case と failure scenario を使って検証します。`v1alpha1` は壊してよいという意味ではなく、利用者と migration を調整する責任があります。

## community roadmap

- issue で use case と decision を公開。
- 月次 community meeting と非同期 proposal。
- reviewer / maintainer promotion criteria。
- 企業名ではなく個人の role と conflict policy。
- security contact と release manager を複数人化。
- adopter feedback を roadmap に反映しつつ、1 社専用機能に偏らない。

