# package と API 設計

## package は変更理由で分ける

`models`、`utils`、`common` のような技術カテゴリは成長すると依存の集積地になります。`task`、`scheduler`、`lease` のような問題領域で名前を付け、public API を小さく保ちます。

推奨されるよくある配置:

```text
cmd/<binary>/       process の組み立てと起動
internal/<domain>/  module 内だけで使う domain/application code
api/                protobuf/OpenAPI など安定させる契約（必要な場合）
```

これは規則ではありません。既存リポジトリを流行の layout に作り替えるのではなく、依存方向と ownership が明瞭かで判断します。

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

## API review の質問

- 利用者が誤用しにくいか。
- zero value は有効か。無効なら constructor で保証できるか。
- キャンセルと timeout は伝播するか。
- 並行利用は安全か。docs に明記したか。
- error を機械的に分類できるか。
- 破壊的変更なしで将来拡張できるか。
- test のためだけでなく、本番上の境界を表しているか。

