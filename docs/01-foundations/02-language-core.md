# Go 言語の核

## 値、zero value、型

変数は必ず型を持ち、初期化しなければ zero value になります。`int` は `0`、`bool` は `false`、pointer / slice / map / channel / function / interface は `nil` です。zero value で安全に使える型を設計すると、constructor の強制を減らせます。`sync.Mutex`、`bytes.Buffer` は代表例です。

```go
type Counter struct {
    mu sync.Mutex
    n  int
}

func (c *Counter) Add() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.n++
}
```

`Counter{}` はそのまま使えます。ただし mutex を使用後にコピーしてはいけません。

## array、slice、map

- array `[3]int` は長さも型の一部で、値としてコピーされる。
- slice `[]int` は backing array の一部分を表す descriptor。`len` と `cap` がある。
- map は hash map への参照的な値。読み書きを複数 goroutine から無同期で行えない。

slice を append すると容量次第で backing array が変わります。呼び出し元へ長さの変更を返したいなら、更新後の slice を返します。

```go
func appendValid(dst []string, values ...string) []string {
    for _, v := range values {
        if v != "" {
            dst = append(dst, v)
        }
    }
    return dst
}
```

## pointer と method set

pointer は「共有可能なアドレス」であって、自動的に高速という意味ではありません。

pointer receiver を選ぶ主な理由:

- receiver の状態を変更する。
- mutex などコピー禁止の値を持つ。
- 大きな構造体のコピーを避ける。
- method set を一貫させる。

小さく不変な値（時刻や座標など）は value receiver が自然な場合があります。同じ型で混在させる前に理由を持ちます。

## defer

`defer` は現在の関数が return するとき LIFO 順で実行されます。resource を取得した直後に解放を予約すると、途中の return に強くなります。

```go
f, err := os.Open(name)
if err != nil {
    return err
}
defer f.Close()
```

loop 内で大量に defer すると関数終了まで resource が残ります。1 iteration を小さな関数に分けます。

## generics

generics は「複数の型で同じアルゴリズムを、型安全に再利用する」ために使います。単に implementation を隠す目的なら interface の方が適切なことがあります。

```go
func Map[S ~[]E, E any, R any](in S, f func(E) R) []R {
    out := make([]R, len(in))
    for i, v := range in {
        out[i] = f(v)
    }
    return out
}
```

制約を必要以上に抽象化せず、まず具体的なコードを 2〜3 箇所書いて重複の形が見えてから導入します。

## チェックポイント

- `nil` slice と空 slice はどこが同じで、JSON ではどう異なり得るか。
- slice の要素を書き換えたとき呼び出し元にも見える理由は何か。
- pointer receiver が必要なケースを 3 つ挙げられるか。
- zero value で安全な自作型を 1 つ設計できるか。

