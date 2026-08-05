# Go Cloud Native Study Archive

Go を初めて学ぶところから、世界中で長期運用されるクラウドネイティブ OSS を設計・実装・運営できるところまでをつなぐ、日本語の学習ナレッジベースです。

> CNCF の `Graduated` は「高度なコードを書いた」だけで得られる称号ではありません。採用実績、健全なガバナンス、複数組織による保守、セキュリティ、リリース品質などを含むプロジェクト全体の成熟度です。本教材ではコードと同じくらい、運用とコミュニティも学びます。

## 最初の進み方

1. [学習ロードマップ](ROADMAP.md) で現在地と到達基準を確認する
2. [環境構築と Go ツールチェーン](docs/01-foundations/01-toolchain.md) を実行する
3. [Go の核となる考え方](docs/01-foundations/02-language-core.md) から順に読む
4. 各章のチェックポイントを自分の言葉で説明する
5. `go test ./...` を実行し、[サンプルサービス](cmd/taskapi/main.go) を改造する
6. [実在リポジトリの読み方](docs/04-repository-guides/README.md) に進む
7. [最終プロジェクト](docs/05-projects/capstone.md) を小さな段階に分けて作る

## コンテンツ地図

| 領域 | 内容 | 実践物 |
|---|---|---|
| [基礎](docs/01-foundations/README.md) | ツール、型、関数、interface、error、モジュール | 小さな関数とテーブル駆動テスト |
| [Go エンジニアリング](docs/02-engineering/README.md) | 並行処理、HTTP、テスト、設計、性能 | in-memory Task API |
| [クラウドネイティブ](docs/03-cloud-native/README.md) | コンテナ、Kubernetes、分散システム、可観測性、セキュリティ | production 化チェックリスト |
| [OSS コードリーディング](docs/04-repository-guides/README.md) | Kubernetes、containerd、Prometheus、etcd | 読解ノートと小さな contribution |
| [プロジェクト](docs/05-projects/README.md) | 段階別演習と capstone | ローカルツールから分散コントロールプレーンへ |
| [用語集](docs/glossary.md) | Go / 分散システム / CNCF の重要語 | 復習用インデックス |
| [技能チェック](docs/checklist.md) | 各段階の実演可能な到達条件 | commit・PR・実験記録 |
| [参考資料](docs/references.md) | 一次情報と公式リンク | 深掘りの入口 |

## 教材コード

必要なのは Go 1.22 以降です。外部依存を使わず、標準ライブラリの設計を学べるようにしています。

```console
go test ./...
go test -race ./...
go vet ./...
go run ./cmd/taskapi
```

`go test -race` は環境によって C toolchain が必要です。Windows で `C compiler not found` になる場合は、対応 compiler を導入するか Linux の CI / container で実行してください。race 検査を省略してよいという意味ではありません。

別ターミナルで動作を確認します。

```console
curl -i http://localhost:8080/healthz
curl -i http://localhost:8080/readyz
curl -i -X POST http://localhost:8080/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{"title":"learn context"}'
curl -i http://localhost:8080/v1/tasks
```

Windows PowerShell では `curl.exe` を使うと同じ例を実行できます。

## 学び方のルール

- 写経で終えず、必ず入力・失敗・キャンセルの条件を変える。
- goroutine を作ったら、誰が停止させ、誰が待つかを説明する。
- API を作ったら、タイムアウト、互換性、メトリクス、認可を考える。
- ベンチマークの数値より先に、要件と計測条件を書く。
- 大きな OSS は最初から全体を読まず、1 回のリクエスト経路を追う。
- 学習記録は「知ったこと」ではなく「判断できるようになったこと」で残す。

## この教材の完成条件

最終的に次を自力で説明・実演できれば、Go の学習からプロダクト開発へ移る準備ができています。

- API の境界に小さな interface を置き、具体型とエラー方針を説明できる。
- `context.Context` によるキャンセルと、リークしない goroutine を設計できる。
- race detector、fuzz、benchmark、profile を目的に応じて使い分けられる。
- 冪等性、一貫性、リトライ、バックプレッシャー、リーダー選出を説明できる。
- SLI/SLO、構造化ログ、メトリクス、トレースを運用判断につなげられる。
- 脅威モデル、依存管理、署名、SBOM、脆弱性対応をリリース工程に組み込める。
- 公開ロードマップ、互換性方針、意思決定、メンテナ育成を設計できる。
