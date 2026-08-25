# 用語集

短い定義と「なぜ重要か」をセットにしています。章を読んだ後、自分の例を追記してください。

## Go

| 用語 | 意味 |
|---|---|
| goroutine | Go runtime が多重化する軽量な実行単位。安価でも寿命管理は必要。 |
| channel | goroutine 間で値を送受信し、同期も行う typed conduit。queue の万能代替ではない。 |
| context | request scope の deadline、cancel signal、限定的 metadata を伝播する契約。optional parameter bag ではない。 |
| cancellation | 処理へ停止を要求する協調的な signal。停止完了そのものではなく、対象が signal を観測して return した後に join が必要。 |
| join | 開始した goroutine や処理の終了を owner が待ち、結果や error を回収すること。channel receive や `WaitGroup.Wait` などで表す。 |
| data race | 同じ memory に並行アクセスし、少なくとも一方が write で同期がない状態。結果の論理的競合（race condition）とは区別する。 |
| escape analysis | 値を stack に置けるか heap に逃がすかを compiler が解析すること。最適化前に計測する。 |
| interface | method set による振る舞いの契約。実装宣言を要求しない structural typing。 |
| method set | ある型または pointer 型が持つ method の集合。interface を満たすかに影響する。 |
| zero value | 明示初期化されていない値の既定値。すぐ使える型は扱いやすい。 |
| slice | backing array を参照する、pointer・length・capacity 相当の descriptor。 |
| aliasing | 複数の値が同じ memory を参照すること。一方の変更が他方から見えるため、API 境界では所有権または copy 方針が必要。 |
| defensive copy | 呼び出し側との意図しない alias を断つため、保持する入力や返す mutable value をコピーすること。安全性と allocation cost を比較する。 |
| rune | Unicode code point を表す `int32` の alias。文字表示単位（grapheme）とは限らない。 |
| defer | 現在の関数終了時に呼び出しを実行する仕組み。resource cleanup に有効。 |
| panic / recover | 通常の error return では扱わない異常な control flow と、deferred function での捕捉。 |
| build tag | OS、architecture、feature 等で build 対象 file を選ぶ constraint。 |
| internal package | 親の subtree 外から import できないことを Go tool が強制する package。 |
| sentinel error | package が公開する比較可能な既知の error 値。`errors.Is` で分類できるが、公開後は API 契約になる。 |
| error chain | `%w` や `Unwrap` で文脈付き error から原因をたどれる関係。表示文字列ではなく `errors.Is/As` で調べる。 |
| table-driven test | 入力・期待値・名前を表にし、同じ検証を subtest で繰り返す Go で一般的な形式。 |
| fuzzing | 生成・変異した入力で crash や invariant 違反を探索する test 手法。 |

## API・分散システム

| 用語 | 意味 |
|---|---|
| idempotency（冪等性） | 同じ操作を複数回適用しても、1 回と同じ最終状態になる性質。retry 安全性の中心。 |
| linearizability | 各操作が呼び出しと応答の間の一点で原子的に起きたように見える強い整合性。 |
| eventual consistency | 更新が止まれば replica が最終的に同じ値へ収束するモデル。いつ・どう収束するかの定義が必要。 |
| consensus | 障害がある複数 node が値や log の順序に合意する問題。Raft は代表的 algorithm。 |
| quorum | 合意や read/write に必要な node 数。過半数は quorum 同士が交差する。 |
| lease | 期限付きの権利。clock、renewal failure、fencing を考慮する。 |
| fencing token | 古い lease holder の書き込みを storage 側で拒否する単調増加 token。 |
| WAL | durable state の前に追記する write-ahead log。crash recovery に用いる。 |
| backpressure | consumer の処理能力に合わせて producer を抑制・拒否する仕組み。 |
| circuit breaker | 失敗中の依存先への呼び出しを一時遮断し、連鎖障害を抑える状態機械。 |
| retry budget | retry が通常 traffic や依存先を圧迫しないように設ける上限。 |
| thundering herd | 同じ event を契機に多数の処理が一斉起動し、resource を圧迫する現象。jitter が緩和に有効。 |
| reconciliation | desired state と observed state の差を繰り返し埋め、収束させる処理。controller の基本。 |
| at-least-once | message を失わない代わりに重複し得る配送。consumer の冪等性が必要。 |

## 運用・CNCF

| 用語 | 意味 |
|---|---|
| SLI | 可用性や latency など、利用者体験を表す計測指標。 |
| SLO | SLI に対する期間内の目標値。例: 30日で成功率 99.9%。 |
| error budget | SLO が許容する失敗量。信頼性と変更速度の判断に使う。 |
| RED | Rate、Errors、Duration を service ごとに見る監視方法。 |
| USE | Utilization、Saturation、Errors を resource ごとに見る監視方法。 |
| cardinality | metric label の組み合わせ数。無制限な user ID 等は TSDB の負荷を急増させる。 |
| graceful shutdown | 新規 traffic を止め、有限 deadline 内で in-flight work を完了させてから終了する手順。deadline 超過時の強制終了方針も必要。 |
| EndpointSlice | Service の backend endpoint と `ready`、`serving`、`terminating` などの condition を表す API。Pod 削除時の traffic drain を観測する境界。 |
| termination grace period | kubelet が Pod の graceful termination に与える総時間。`preStop` と process 終了の両方が消費し、満了後は強制終了される。 |
| routing propagation | readiness や endpoint の変更が proxy、load balancer、client の経路選択へ反映されるまでの伝播。同期的・瞬時とは限らない。 |
| SBOM | software を構成する component と version の一覧。脆弱性影響調査に使う。 |
| provenance | artifact がどの source と build process から生成されたかを示す情報。 |
| maintainer | project の方向性と統合品質に責任を持つ役割。単なる commit 権限ではない。 |
| governance | 意思決定、role、選出、紛争解決などを明文化した project 運営の仕組み。 |
| CNCF Sandbox | 初期段階の cloud native project が ecosystem 内で育つ entry point。 |
| CNCF Incubating | adoption と技術・community の成熟を示した段階。 |
| CNCF Graduated | 広範な adoption、健全な governance、security と process の成熟を示す段階。 |
