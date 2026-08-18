# 技能チェックリスト

日付と証拠（commit、PR、実験ログ）を添えて使います。読んだだけでは完了にしません。

## Go 基礎

- [ ] slice の backing array 共有を test で示した。
- [ ] 保持する slice の入力・出力で copy が必要か判断し、alias を防ぐ test を書いた。
- [ ] value / pointer receiver の選択理由を説明した。
- [ ] consumer 側の小さな interface と fake を作った。
- [ ] wrapped error を `errors.Is/As` で分類した。
- [ ] store の失敗が service の文脈を保ち、transport の応答へ変換される経路を追った。
- [ ] `io.Reader/Writer` を使って I/O source を交換した。
- [ ] table-driven test と fuzz test を書いた。

## 並行処理

- [ ] goroutine ごとに owner、cancel、join、error path を示した。
- [ ] channel close の owner を説明した。
- [ ] worker concurrency と queue に上限を設けた。
- [ ] `go test -race` で意図的な race を検出し、修正した。
- [ ] deadline と caller cancellation を伝播した。
- [ ] cancel が通知であって完了ではないことを、cancel 後の join test で示した。
- [ ] shutdown 中に readiness を落とし、in-flight request / work の完了と期限超過を test した。

## Service / Production

- [ ] input size と resource consumption を制限した。
- [ ] authentication と resource-level authorization を実装した。
- [ ] schema/API の後方互換 migration を実演した。
- [ ] RED metrics、structured logs、trace を関連付けた。
- [ ] SLI/SLO と burn-rate alert を定義した。
- [ ] pprof で bottleneck を特定し、benchmark で改善を証明した。
- [ ] backup から別環境へ restore した。

## 分散システム

- [ ] retry 可能性と idempotency key を設計した。
- [ ] stale lease holder を fencing token で拒否した。
- [ ] response loss と重複 delivery を注入した。
- [ ] network partition 中の consistency/availability 選択を説明した。
- [ ] version skew を含む rolling upgrade を test した。
- [ ] overload 時に明示的な backpressure / admission control が働いた。

## Security / Supply chain

- [ ] data flow と trust boundary を含む threat model を作った。
- [ ] secret / PII が log、metric、artifact にないことを確認した。
- [ ] dependency vulnerability を到達可能性とともに triage した。
- [ ] release に SBOM、署名、provenance を付けた。
- [ ] private disclosure から patch release まで演習した。
- [ ] least privilege と credential rotation を検証した。

## OSS / Community

- [ ] 実在 OSS の 1 request path と failure test を読解した。
- [ ] 読んだ source の release tag と commit SHA を記録し、後日同じ根拠を再現できる。
- [ ] 複数 process の処理を直列 call graph にせず、owner・watch/queue・state transition を図示した。
- [ ] unit / controller / integration / live の各 evidence が証明しない範囲を説明した。
- [ ] upstream に再現 test または小さな修正を contribution した。
- [ ] 他者の PR を建設的に review した。
- [ ] governance、role、promotion、conflict process を公開した。
- [ ] 複数組織の production adopter から継続 feedback がある。
- [ ] 自分なしで triage、release、security response が動いた。
