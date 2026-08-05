# ツールチェーン

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

## 演習

- `go list -deps ./cmd/taskapi` で標準ライブラリ依存を観察する。
- `go test -run TestCreate -v ./internal/task` のように 1 テストだけ実行する。
- `go test -count=20 ./...` で非決定的なテストがないか探す。
- `go test -race ./...` と通常の test の違いを自分の言葉で書く。

