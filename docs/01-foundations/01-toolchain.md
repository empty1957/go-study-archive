# ツールチェーン

> この章の問い: 変更が「動いた」ではなく「境界の契約を満たした」と、どのコマンドで示すか。

## module と package

- **module**: `go.mod` をルートに持つバージョン管理単位。import path の接頭辞を宣言する。
- **package**: 同じディレクトリで一緒にコンパイルされる `.go` ファイル群。
- **command**: `package main` と `func main()` を持つ実行可能プログラム。

本リポジトリでは `cmd/taskapi` が command、`internal/task` が外部 module から import できない内部 package です。

## 毎日使うコマンド

```console
go env                 # 有効な環境設定
go doc net/http.Server # 端末で API を調べる
go list ./...          # module 内 package の列挙
go fmt ./...           # 標準の整形
go test ./...          # 全 package のテスト
go test -race ./...    # data race の動的検出
go test -cover ./...   # coverage（品質そのものではない）
go vet ./...           # コンパイル可能だが怪しいコードを検査
go mod tidy            # import と依存宣言を同期
go build ./cmd/taskapi # command をコンパイル
```

## バージョンと依存

`go.mod` は直接・間接依存と Go language version を管理し、`go.sum` は取得した module 内容の検証に使われます。ライブラリは SemVer を意識し、`v2` 以降では通常 import path に `/v2` が入ります。

依存を増やす前に次を確認します。

1. 標準ライブラリで十分か。
2. API と保守状態は安定しているか。
3. license と supply-chain risk を許容できるか。
4. transitive dependency はどれだけ増えるか。
5. 削除・置換できる境界に置けるか。

## コンパイラを先生にする

Go は unused import / variable をエラーにします。エラーの最初の位置から直し、変更ごとに `go test` を回します。エディタの自動修正だけで終わらず、`go doc` で型と契約を確認してください。

## 変更から証拠までの短いループ

コマンドは全部を無条件に回すのではなく、知りたいことに合わせて狭い検証から広げます。

| 段階 | コマンド | 分かること | 分からないこと |
|---|---|---|---|
| 対象を知る | `go list ./...` | build 対象の package | 実行時の正しさ |
| 速く仮説を確認 | `go test -run TestServicePreservesNotFound ./internal/task` | 1 つの error 契約 | 他 package への影響 |
| package 契約を確認 | `go test ./internal/task` | 対象 package の test | data race、未検査 package |
| module 全体を確認 | `go test ./...` | 全 package の test と compile | test が通らない入力 |
| 静的な怪しさを探す | `go vet ./...` | vet が知る誤用パターン | 一般的な仕様違反のすべて |
| 並行アクセスを実行時検査 | `go test -race ./...` | 実行された経路の data race | 実行されなかった経路 |

`go fmt`、`go test`、`go vet` は代替関係ではありません。整形、振る舞い、静的検査という別の証拠です。失敗したら、コマンド、Go version、最初の error、再現対象 package を学習ログに残します。

### よくある失敗と切り分け

- `no required module provides package`: import path と `go.mod` の module path を照合する。
- `found packages x and y`: 同じ directory に異なる package 宣言が混ざっていないか確認する。
- test が単独では通るが全体では落ちる: global state、時刻、順序、port、並行実行への依存を疑う。
- race detector を起動できない: code が安全だと結論せず、C toolchain 不足など環境条件を記録して別環境で補う。

## 演習

- `go list -deps ./cmd/taskapi` で標準ライブラリ依存を観察する。
- `go test -run TestServicePreservesNotFound -v ./internal/task` で 1 つの境界契約だけを実行する。
- `go test -run ExampleService_Get_errorContract -v ./internal/task` で実行可能な例を確認する。
- `go test -count=20 ./...` で非決定的なテストがないか探す。
- `go test -race ./...` と通常の test の違いを自分の言葉で書く。

---

[セクション概要](README.md) | 次: [言語の核](02-language-core.md)
