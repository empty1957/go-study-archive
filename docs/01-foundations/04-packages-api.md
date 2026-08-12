# package と API 設計

> この章の問い: 値と失敗の契約を、変更理由が混ざらない package 境界へどう配置するか。

## package は変更理由で分ける

`models`、`utils`、`common` のような技術カテゴリは成長すると依存の集積地になります。`task`、`scheduler`、`lease` のような問題領域で名前を付け、public API を小さく保ちます。

推奨されるよくある配置:

```text
cmd/<binary>/       process の組み立てと起動
internal/<domain>/  module 内だけで使う domain/application code
api/                protobuf/OpenAPI など安定させる契約（必要な場合）
```

これは規則ではありません。既存リポジトリを流行の layout に作り替えるのではなく、依存方向と ownership が明瞭かで判断します。

## Task API の変更経路を一方向にする

このリポジトリでは [`cmd/taskapi`](../../cmd/taskapi/main.go) が process を組み立て、[`internal/task`](../../internal/task) が domain と use case を持ちます。

```text
cmd/taskapi (composition root)
    |
    v
internal/task/http.go  ->  service.go  ->  Store interface
       transport            use case          boundary
                                                ^
                                                |
                                           MemoryStore
```

矢印は compile-time dependency です。`service.go` が `cmd/taskapi` や HTTP response 型を import しないため、CLI や worker からも同じ use case を呼べます。`MemoryStore` を作る場所を `main` に置くことで、domain 側が concrete storage の lifecycle を所有せずに済みます。

新しい要件を配置するときは「どの技術を使うか」より先に、変更理由を分類します。

| 変更 | 主な配置 | 判断理由 |
|---|---|---|
| title は 200 文字以内 | service/domain | transport が変わっても維持する rule |
| 不明な JSON field を拒否 | HTTP handler | JSON input の契約 |
| not-found を 404 にする | HTTP handler | domain error から HTTP への変換 |
| map を mutex で守る | store implementation | concrete storage の並行利用 |
| server timeout を決める | `cmd/taskapi` | process の運用設定 |

## public API のコスト

大文字で始まる identifier は package 外へ export されます。一度利用者が持つと、変更には migration が必要です。

- concrete struct の field を公開するか、constructor / method にするか。
- option を増やす可能性があるなら config struct や functional option が妥当か。
- interface を引数で受け、戻り値では具体型を返せるか。
- context は通常第 1 引数。struct field に保存しない。
- caller が resource の close を担当するか明記する。

## domain と transport を分ける

HTTP status や JSON tag を domain 判断の中心に入れると、gRPC や worker から再利用しづらくなります。

```text
HTTP handler -> application service -> repository
     JSON         domain error          storage error
```

handler は decode / validate / error mapping / encode、service は use case、repository は保存の詳細を担当します。小規模なうちはファイル数を増やしすぎず、この境界だけ守ります。

## dependency injection

DI framework がなくても、constructor で依存を明示できます。

```go
type Service struct { store Store }

func NewService(store Store) *Service {
    return &Service{store: store}
}
```

`main` は具体 implementation を組み立てる composition root です。domain package が SQL driver や HTTP router を勝手に生成しないようにします。

## 境界を増やすコストと判断順序

package、interface、constructor はいずれも変更を局所化できますが、名前と移動経路を増やします。小さいコードを最初から層ごとに別 package へ分割すると、循環依存を避けるためだけの型変換や「どこにあるか探す時間」が増えます。

次の順で判断します。

1. 現在の変更理由が本当に二つ以上あるか。
2. consumer が必要とする契約を、具体型の method だけで十分に表せるか。
3. I/O、process、protocol など寿命や失敗方針が変わる地点か。
4. 境界を test すると、production の判断も明確になるか。
5. 分割後の依存矢印を一方向に保てるか。

Task API は学習用に同じ `task` package 内でファイルを分けています。規模が増えたという理由だけで package を増やさず、transport と use case が別の cadence や owner を持つようになった時点で分割を検討します。

## API review の質問

- 利用者が誤用しにくいか。
- zero value は有効か。無効なら constructor で保証できるか。
- キャンセルと timeout は伝播するか。
- 並行利用は安全か。docs に明記したか。
- error を機械的に分類できるか。
- 破壊的変更なしで将来拡張できるか。
- test のためだけでなく、本番上の境界を表しているか。

## セクション演習: 一つの変更を端から端まで設計する

Task に label を追加すると仮定し、実装前に次を 1 ページへ書きます。

1. `[]string` を誰が所有し、保存時・返却時のどこで copy するか。
2. 空 label、重複、上限超過を domain error としてどう分類するか。
3. `Store` の method を変更する必要があるか。不要なら理由は何か。
4. HTTP JSON と domain value の変換をどこに置くか。
5. alias、validation、error mapping をそれぞれどの test で固定するか。

[言語の核](02-language-core.md) の copy 判断と [interface と error](03-interfaces-errors.md) の error chain を同じ設計に適用できれば、このセクションの出口です。[技能チェックリスト](../checklist.md#go-基礎) に設計と実験 commit を記録してください。

---

前: [interface と error](03-interfaces-errors.md) | [セクション概要](README.md) | 次: [Go エンジニアリング](../02-engineering/README.md)
