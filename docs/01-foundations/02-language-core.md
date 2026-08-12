# Go 言語の核

> この章の問い: 値を渡したとき、何がコピーされ、どの変更が呼び出し元から見えるか。

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

## 「値渡し」と「共有しない」は同じではない

Go の代入、引数、戻り値は値をコピーします。ただし、コピーされた値が pointer、slice、map などを含めば、その先の storage は共有され得ます。「値渡しだから安全」ではなく、**何がコピーされ、何を指しているか**まで追います。

| 型 | 代入でコピーされるもの | コピー後に共有し得るもの | API 境界での判断 |
|---|---|---|---|
| `int`、`bool`、要素も値だけの struct | 値全体 | なし | 小さければ value が自然 |
| pointer | address | 指し先 | nil、寿命、変更権限を決める |
| slice | pointer・length・capacity 相当 | backing array | 入出力を保持するなら copy を検討 |
| map | map descriptor | entry 群 | 所有者と並行利用を明示 |
| `sync.Mutex` を含む struct | struct 全体 | lock が守る状態との対応が壊れる | 使用開始後はコピー禁止、pointer で扱う |

このリポジトリの [`Task`](../../internal/task/task.go) は field が安全にコピーできる値だけなので、`MemoryStore.Get` は `Task` を値で返せます。一方、将来 `[]string` の label を追加するなら、保存時と返却時のどちらで copy するかを API 契約として決める必要があります。

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

### slice の alias を境界で断つ

次の constructor は caller の slice を保持するため、caller が後から要素を書き換えると内部状態も変わります。

```go
func NewChecklist(items []string) Checklist {
    return Checklist{items: items} // backing array を共有する
}
```

境界の内側が状態を所有する契約なら、入力と出力の両方で copy します。

```go
func NewChecklist(items []string) Checklist {
    return Checklist{items: append([]string(nil), items...)}
}

func (c Checklist) Items() []string {
    return append([]string(nil), c.items...)
}
```

実行可能な全体は [`examples/foundations`](../../examples/foundations/ownership.go) にあります。`go test -run TestChecklistOwnsItems ./examples/foundations` を実行し、constructor 側または getter 側の copy を外して、どちらの assertion が落ちるか確認してください。

copy には allocation と転送量のコストがあります。常に copy するのではなく、値が小さい、境界をまたいで保持する、外部からの変更を許したくない場合に選びます。大きな byte buffer のように copy が高価なら、所有権移譲を明記する、immutable な view を返す、処理を callback の寿命内に限定するといった設計も比較します。

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
- slice を field に保持する API で、入力時だけ copy しても十分でないのはなぜか。
- pointer receiver が必要なケースを 3 つ挙げられるか。
- zero value で安全な自作型を 1 つ設計できるか。

---

前: [ツールチェーン](01-toolchain.md) | [セクション概要](README.md) | 次: [interface と error](03-interfaces-errors.md)
