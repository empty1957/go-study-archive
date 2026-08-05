# interface と error

## interface は利用側で小さく定義する

Go の interface は暗黙的に満たされます。implementation package が巨大な interface を公開するより、利用側が必要な振る舞いだけを宣言します。

```go
type TaskFinder interface {
    Find(ctx context.Context, id string) (Task, error)
}
```

これにより file、database、test fake のどれでも契約を満たせます。interface を「mock を作るためだけ」に増やさず、複数 implementation または境界が本当にあるか考えます。

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

