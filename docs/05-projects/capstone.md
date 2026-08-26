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
- artifact digest と migration compatibility を固定した canary。
- 事前に閾値を決めた promote / hold / rollback gate。
- upgrade、rollback、disaster recovery drill と所要時間の記録。

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

## milestone を閉じる証拠

milestone は「実装した」で閉じず、次の因果関係を第三者が再実行できたときに閉じます。

```text
利用者の failure scenario
        ↓
守る invariant と許容する劣化
        ↓
観測可能な signal と評価 window
        ↓
閾値を越えたときの操作
        ↓
test / drill / production observation
```

たとえば「重複実行を防ぐ」だけでは判定不能です。response loss 後に同じ idempotency key で再送し、job ID が一つであることを integration test で観測します。一方、queue depth や latency は安全 invariant ではなく運用 budget です。短時間の揺らぎを即時 rollback にするか、一定 window の burn rate で判定するかを分けます。

| 証拠 | 問うこと | 不足・違反時の既定動作 |
|---|---|---|
| invariant test | 絶対に許容しない状態を検出できるか | 即時 `Rollback`、または side effect 停止 |
| compatibility test | N-1 client / worker と同時稼働できるか | rollout 開始前に `Hold` |
| 最小 sample / window | 偶然の成功・失敗を判断していないか | 十分になるまで `Hold` |
| SLI threshold | 利用者影響が事前の budget 内か | `Rollback` または traffic 縮小 |
| recovery drill | binary 以外の data / side effect も復旧できるか | rollout 開始前に `Hold` |

判定順序は重要です。invariant 違反は sample 不足より優先します。前提条件が欠けた候補は traffic に出しません。正常な少数 sample は promote せず、十分な sample で全 budget を満たした場合だけ次段階へ進めます。[rollout gate の Go 例](../../examples/rolloutgate/gate.go) と [table test](../../examples/rolloutgate/gate_test.go) を実行し、境界値を変更してこの順序を壊す test を追加してください。

実際の release では candidate と control を同じ window で比較し、deploy 以外の traffic mix や依存先障害を切り分けます。単純な固定閾値は教材用の最小モデルであり、低 traffic service、季節性、複数 SLI では比較方法と観測期間を設計し直します。

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

## release 演習: generation 2

`timeoutSeconds` の省略時既定値を変更する generation 2 を仮定します。

1. N-1 client、generation 1 の実行中 job、generation 2 の新規 job を含む compatibility matrix を書く。
2. immutable な commit / image digest と migration version を evidence packet に記録する。
3. 1% traffic で開始し、最小 sample、評価 window、error / latency budget、invariant を rollout 前に固定する。
4. response loss、stale lease result、queue saturation を注入する。
5. `Promote / Hold / Rollback` の理由と観測 link を記録する。
6. old binary へ戻した後も generation 2 data を読めることを確認する。読めないなら rollback ではなく roll forward または traffic stop を復旧策にする。

よくある失敗は、最新 5 分の平均だけを見て低 traffic の候補を promote すること、alert が鳴ってから rollback 手順を考えること、Deployment の revision を戻せば database も戻ると仮定することです。判定 logic、復旧操作、data compatibility を別々に test します。

## community roadmap

- issue で use case と decision を公開。
- 月次 community meeting と非同期 proposal。
- reviewer / maintainer promotion criteria。
- 企業名ではなく個人の role と conflict policy。
- security contact と release manager を複数人化。
- adopter feedback を roadmap に反映しつつ、1 社専用機能に偏らない。

## 一次資料

- [Google SRE Workbook: Canarying Releases](https://sre.google/workbook/canarying-releases/): candidate / control、評価 window、自動判定を設計する背景。
- [Kubernetes Deployments](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/): rollout の進行状態、revision、Pod template の rollback 範囲。
- [Go: Add a test](https://go.dev/doc/tutorial/add-a-test): `go test` で失敗を再現する最小手順。
