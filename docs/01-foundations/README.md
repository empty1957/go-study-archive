# 01: Go の基礎

このセクションでは `internal/task` の Task API を一本のケーススタディとして追います。文法項目を個別に暗記するのではなく、次の因果関係を説明できることが出口です。

```text
値の性質を知る
  -> 所有者と変更可能範囲を決める
  -> package 境界を置く
  -> interface と error で境界の契約を表す
  -> test で契約を固定する
```

## 読む順序

1. [ツールチェーン](01-toolchain.md)
2. [言語の核](02-language-core.md)
3. [interface と error](03-interfaces-errors.md)
4. [package と API 設計](04-packages-api.md)

各章では同じ問いを一段ずつ具体化します。

| 問い | このリポジトリで見る場所 | 証拠 |
|---|---|---|
| 値の所有者は誰か | [`Task`](../../internal/task/task.go)、[`MemoryStore`](../../internal/task/store.go) | copy / alias の test |
| 失敗を誰が判断するか | [`Service`](../../internal/task/service.go)、[HTTP handler](../../internal/task/http.go) | `errors.Is` と status の test |
| 変更の影響範囲はどこか | `cmd/taskapi -> internal/task` の依存方向 | `go list` と package test |

## 進め方

1. 章のコードを読む前に、入力・出力・所有者・失敗の分類を予想する。
2. 記載された `go test -run ...` を実行し、予想と結果を比較する。
3. 失敗例へ変更して test が落ちることを確認し、元に戻して理由を書く。
4. [技能チェックリスト](../checklist.md#go-基礎) に commit または実験ログを残す。

この段階の目標は、「値の所有者は誰か」「失敗を誰が判断するか」「変更の影響範囲はどこか」を、実行結果とコードで説明できることです。
