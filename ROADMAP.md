# 学習ロードマップ

期間は目安です。週数ではなく「出口条件」を満たしたら次へ進みます。週 8〜10 時間なら、基礎から最初の公開プロダクトまで約 9〜15 か月を想定します。CNCF Graduated はその後の複数年の採用・運営の成果です。

## Phase 0: 開発環境（1週）

学ぶこと: `go` コマンド、module、package、Git、デバッガ、公式ドキュメントの引き方。

出口条件:

- `go mod init`、`go test ./...`、`go fmt ./...`、`go vet ./...` の役割を説明できる。
- コンパイルエラーを読んで、小さな CLI を修正できる。
- 変更理由が 1 つの小さな Git commit を作れる。

## Phase 1: Go の基礎（4〜6週）

学ぶこと: 値とポインタ、slice / map、method、interface、error、defer、I/O、テスト。

作るもの: JSON ファイルを読み書きするタスク管理 CLI。

教材内の実験: [基礎セクション](docs/01-foundations/README.md) で Task API の値の所有権、保存 interface、error chain を端から端まで追い、実行可能な例を変更する。

出口条件:

- zero value、slice の長さと容量、map の並行利用上の制約を説明できる。
- error を値として扱い、`errors.Is/As` で分類できる。
- consumer 側に小さな interface を定義し、テーブル駆動テストを書ける。
- mutable な入力を保持・返却する API で、alias と copy cost のどちらを選ぶか説明できる。

## Phase 2: サービス開発（6〜8週）

学ぶこと: `net/http`、context、goroutine、channel、mutex、graceful shutdown、構造化ログ。

作るもの: このリポジトリの Task API を永続化し、認証とページングを追加する。

教材内の実験: [Go エンジニアリング](docs/02-engineering/README.md)で bounded pipeline と HTTP server に共通する owner → cancel → join → error 回収の経路を追い、shutdown 中の readiness と in-flight work を test する。

出口条件:

- すべての外部 I/O に deadline がある。
- `go test -race ./...` が通る。
- readiness と liveness の違いを実装で示せる。
- 正常系・不正入力・競合・キャンセルをテストできる。
- goroutine ごとの owner、cancel、join、error path を図と実行結果で説明できる。

## Phase 3: Production engineering（8〜12週）

学ぶこと: SQL、migration、キャッシュ、queue、OpenTelemetry、Prometheus、profile、負荷試験、CI/CD。

作るもの: コンテナ化した API、worker、PostgreSQL、メトリクス、ダッシュボード。

出口条件:

- SLO と error budget を定義し、アラートを症状ベースで設計できる。
- pprof と benchmark を使い、再現可能な性能改善を 1 件行える。
- schema/API の後方互換な rollout と rollback を実施できる。

## Phase 4: 分散システム（10〜16週）

学ぶこと: consensus、replication、partition、lease、watch、冪等性、backpressure、rate limit。

作るもの: 複数ノードのジョブスケジューラ、または宣言的 controller。

出口条件:

- ネットワーク分断、重複配送、時計のずれ、部分障害をテストで注入できる。
- at-most-once / at-least-once の選択と、利用者への影響を説明できる。
- reconcile loop が収束する条件と、処理の冪等性を説明できる。

## Phase 5: OSS contribution（並行して継続）

学ぶこと: issue triage、design proposal、code review、release note、互換性、community governance。

教材内の実験: [Kubernetes の repository guide](docs/04-repository-guides/kubernetes.md)で Pod 削除を題材に、固定した release の source、owner、非同期境界、unit/controller test、cluster 観測を一つの evidence chain にする。

進め方:

1. ドキュメントまたは再現テストの修正
2. good first issue
3. bug fix と回帰テスト
4. 小さな機能と設計提案
5. review、triage、release の支援

出口条件:

- user-visible な invariant から source と test を逆引きし、tag / commit SHA 付きの読解ノートを作れる。
- unit test が証明することと、integration test / live observation が必要な仮説を分けられる。
- 複数の upstream 変更を完了し、他者の変更をレビューできる。
- project の contribution guide と意思決定過程に沿って提案できる。
- 自分以外の contributor が成功するためのドキュメントを改善できる。

## Phase 6: 自分のプロダクト（6〜18か月）

[capstone](docs/05-projects/capstone.md) を参照し、狭く重要な問題から始めます。

出口条件:

- 明確な problem statement と非目標がある。
- API compatibility、security、observability、upgrade、backup の方針がある。
- 実利用者から学ぶ公開フィードバックループがある。
- 1 社・1 人に依存しない maintainer / reviewer の経路がある。

## Phase 7: CNCF maturity（複数年）

Sandbox → Incubating → Graduated は学習コースの試験ではなく、プロジェクトの実績です。[Graduated への道](docs/03-cloud-native/06-cncf-product-maturity.md) のチェックリストを半年ごとに評価します。

## 週次サイクル

| 曜日/枠 | 活動 | 成果物 |
|---|---|---|
| 2時間 | 章を読み、例を改造 | 疑問と仮説 |
| 3時間 | 小機能を実装 | 動く commit |
| 2時間 | テスト、race、profile | 計測結果 |
| 2時間 | upstream の PR を読む | コード読解ノート |
| 1時間 | 振り返り | 次週の 1 つの重点 |

## 学習ログのテンプレート

```markdown
# YYYY-MM-DD: テーマ

## 判断できるようになったこと
## 実験したコードと結果
## 失敗した仮説
## 実在 OSS で見つけた例
## まだ説明できないこと
## 次の一手（1つだけ）
```
