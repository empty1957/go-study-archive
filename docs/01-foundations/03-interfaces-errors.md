# interface と error

> この章の問い: 境界の内側で起きた失敗を、実装詳細を漏らさず利用側が判断できる形でどう運ぶか。

## interface は利用側で小さく定義する

Go の interface は暗黙的に満たされます。implementation package が巨大な interface を公開するより、利用側が必要な振る舞いだけを宣言します。

```go
type TaskFinder interface {
    Find(ctx context.Context, id string) (Task, error)
}
```

これにより file、database、test fake のどれでも契約を満たせます。interface を「mock を作るためだけ」に増やさず、複数 implementation または境界が本当にあるか考えます。

### Task API で境界をたどる

実装では [`Service`](../../internal/task/service.go) が consumer、[`Store`](../../internal/task/store.go) が必要最小限の保存契約です。依存の流れは次のようになります。

```text
HTTP handler -> Service -> Store interface <- MemoryStore
     404      errors.Is       ErrNotFound
```

`Service` は `MemoryStore` を知らず、`main` が具体実装を注入します。将来 SQL store を追加しても service の use case は維持できます。一方、実装が一つで差し替え境界もない小さな関数に interface を置くと、読む型が増えるだけです。interface は「実装を隠す道具」ではなく、consumer が必要とする振る舞いを表す境界として導入します。

### interface の nil 罠

interface 値は `(dynamic type, dynamic value)` の組です。typed nil pointer を入れると interface 自体は nil ではありません。

```go
var p *bytes.Buffer = nil
var w io.Writer = p
fmt.Println(w == nil) // false
```

nil を返すなら、具体的な nil pointer を interface に詰めて返さないようにします。

## error は制御可能な情報

通常の失敗に panic を使いません。文脈を追加して返します。

```go
item, err := repo.Find(ctx, id)
if err != nil {
    return Task{}, fmt.Errorf("find task %q: %w", id, err)
}
```

`%w` で原因を保持すると、呼び出し側は `errors.Is` と `errors.As` で判断できます。

```go
if errors.Is(err, task.ErrNotFound) {
    // HTTP 404 へ変換
}
```

エラー文字列を比較して分岐してはいけません。公開 API では sentinel error や型付き error が互換性契約になる点にも注意します。

### error chain の因果関係

Task API の not-found は層ごとに次の役割へ変わります。

| 層 | 行うこと | 行わないこと |
|---|---|---|
| store | `ErrNotFound` という分類を返す | HTTP status を決めない |
| service | `get task "id"` という操作文脈を `%w` で加える | 原因を文字列だけに潰さない |
| handler | `errors.Is(err, ErrNotFound)` で 404 に変換する | store の実装型で分岐しない |
| process boundary | request 情報と最終的な未処理 error を一度記録する | 各層で同じ失敗を重複記録しない |

`fmt.Errorf("get task: %v", err)` は表示上は似ていますが error chain を切ります。`%w` は caller に原因を判断させるための契約です。反対に、下位の詳細を公開したくない security boundary では、既知の分類へ変換し、元 error は内部 log だけに残す判断も必要です。

実装例を次で確認できます。

```console
go test -run ExampleService_Get_errorContract -v ./internal/task
go test -run TestServicePreservesNotFound -v ./internal/task
```

## どこで log するか

低い層で log し、さらに上へ error を返すと二重記録になりがちです。原則は:

- library / domain: 文脈を付けて返す。
- process boundary（HTTP handler、worker loop、`main`）: request 情報とともに一度記録する。
- retry する層: retry 回数や最終結果を記録する。

## panic の境界

panic は programmer error、破壊された invariant、起動時に続行不能な設定などに限定します。server は request 単位の recovery で process 全体の停止を防げますが、panic の原因を握りつぶしてはいけません。

## 演習

1. `ErrNotFound` を wrap しても `errors.Is` が true になるテストを書く。
2. validation error 型に field 名を持たせ、HTTP 400 に変換する。
3. `io.Reader` を受け取る parser を作り、string/file/network を同じ関数で扱う。
4. 5 method の interface を利用箇所ごとに 1〜2 method に分解する。
5. `Service.Get` の `%w` を `%v` に変えて該当 test を実行し、どの層の契約が壊れたか説明してから戻す。

---

前: [言語の核](02-language-core.md) | [セクション概要](README.md) | 次: [package と API 設計](04-packages-api.md)
