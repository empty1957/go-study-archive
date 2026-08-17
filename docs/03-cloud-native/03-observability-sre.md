# 可観測性と SRE

## telemetry は質問に答えるためにある

- metrics: 集約可能な傾向と alert。何がどれだけ起きたか。
- logs: 離散 event の文脈。なぜこの request が失敗したか。
- traces: service をまたぐ 1 request の因果と latency。
- profiles: process 内の resource 消費。

同じ service / operation / trace の命名規約を揃え、相互に移動できるようにします。

## SLI / SLO

例: 「rolling 30 日で、有効な create request の 99.9% が 500ms 以内に成功する」。分母・成功条件・除外条件・測定点を曖昧にしません。

alert は CPU 70% のような原因候補だけでなく、multi-window burn-rate など user impact と error budget 消費に結びつけます。page には即時の人間対応が有効なものだけを選びます。

## structured logging

安定した key（`service`, `operation`, `request_id`, `trace_id`, `error_kind`, `duration_ms`）を使います。password、token、cookie、個人情報、request body 全体を記録しません。error text を metric label にしないでください。

## metrics の設計

counter、gauge、histogram の意味を守ります。label は bounded set に限定し、user ID、raw URL、error string を入れません。exporter 自身が落ちても core path を block しないようにします。

minimum dashboard:

- request rate / error / latency（RED）。
- CPU / memory / goroutine / GC。
- queue depth / worker saturation / rejected work。
- dependency latency / error / retry。
- build version と rollout marker。

## termination を観測する

graceful shutdown は「error が出なかった」だけでは検証できません。[Pod の終了契約](01-containers-kubernetes.md#pod-削除で並行して起きること)に沿って、少なくとも次を同じ rollout timeline へ載せます。

- termination signal の受信数と時刻。
- readiness が false になった時刻と EndpointSlice condition の変化。
- routing propagation 待機時間、shutdown 開始・完了時間。
- shutdown 開始時の in-flight request 数と、完了・cancel・deadline 超過の内訳。
- forced close / `SIGKILL`、再送、重複 side effect、client error の有無。

Pod が消えるまでの時間だけを短くすると、利用者側の retry や tail latency へ失敗を移すことがあります。rollout marker と request の RED、termination duration を関連付け、通常時と rollout 時の SLI を比較します。Pod UID や request ID は log / trace には有用ですが、無制限な metric label にはしません。

## runbook と incident

alert には owner、impact、確認 query、safe mitigation、escalation、関連 dashboard を付けます。incident 後は個人を責めず、検知・設計・process の改善と owner / deadline を記録します。同じ failure を自動 test に変換することが最も強い学習です。
