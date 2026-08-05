# テスト戦略

## test pyramid ではなく feedback portfolio

| 種類 | 発見するもの | 実行頻度 |
|---|---|---|
| unit | domain rule、edge case | 保存ごと |
| integration | DB、filesystem、protocol の境界 | PR ごと |
| contract | client/server の互換性 | PR / release ごと |
| end-to-end | 配布物を通した主要 user journey | PR / release ごと |
| fuzz | parser、codec、state transition の未知入力 | CI で時間制限付き |
| race | 実行時の同期漏れ | CI / 定期 |
| benchmark | 性能 regression | 条件を固定して定期 |
| failure/chaos | timeout、kill、partition、resource 枯渇 | staging / 定期 |

## table-driven test

```go
tests := []struct {
    name    string
    input   string
    wantErr bool
}{
    {name: "valid", input: "learn Go"},
    {name: "empty", input: "", wantErr: true},
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        // arrange, act, assert
    })
}
```

case 名は失敗した business rule が分かるようにします。内部 field をすべて比較する brittle test より、公開 contract と observable behavior を検証します。

## test double

- fake: 単純だが動く implementation（in-memory store）。
- stub: 決められた response を返す。
- spy: 呼び出しを記録する。
- mock: 期待する interaction を厳密に検証する。

interaction の順番を過度に mock すると refactor に弱くなります。重要な外部 boundary は本物に近い integration test も持ちます。

## fuzz test の invariant

parser なら「panic しない」「成功した値を encode/decode すると同値」「resource 上限を超えない」などを検証します。

```go
func FuzzParse(f *testing.F) {
    f.Add("known-good")
    f.Fuzz(func(t *testing.T, s string) {
        got, err := Parse(s)
        if err == nil && !got.Valid() { t.Fatal("invalid success") }
    })
}
```

## 非決定性を制御する

time、random、network を直接呼ぶと test が遅く不安定になります。ただし何でも interface にせず、clock/function injection や local test server (`httptest`) の最小境界を使います。最終的には real clock/network の integration test も必要です。

## 良い coverage の問い

coverage 80% は目的ではありません。次を確認します。

- 失敗時の cleanup は実行されるか。
- cancel と timeout の直前・直後で安全か。
- 同じ request が重複したらどうなるか。
- partial write / malformed response を扱えるか。
- version N と N+1 は相互運用できるか。
- test 自体が意図どおり失敗することを一度確認したか。

