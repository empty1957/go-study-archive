# ファイル置換を commit point で考える

> この章の問い: process がどの瞬間に止まっても、利用者に「途中まで書いた JSON」を読ませないために、何を commit point にするか。

## 前提と到達点

[言語の核](02-language-core.md)の `defer`、[interface と error](03-interfaces-errors.md)の error chain、[package と API 設計](04-packages-api.md)の I/O 境界を理解してから進みます。ここでは Project A の Task CLI が `tasks.json` 全体を更新する場面を扱います。

到達点は「atomic write」という名前を覚えることではありません。次の保証を分けて説明し、必要なものを選べることです。

| 保証 | 問い |
|---|---|
| 内容の完全性 | 短い write や encode 失敗で途中の bytes を採用しないか |
| 可視性の原子性 | reader は旧版か新版だけを見て、途中版を見ないか |
| durability | 成功を返した直後に電源断しても新版が残るか |
| 同時更新の整合性 | 二つの writer の更新消失を検出できるか |

一つの `os.WriteFile` や `os.Rename` が四つすべてを保証するわけではありません。

## 直接上書きが壊す invariant

既存の `tasks.json` を truncate してから書くと、process が途中で終了した時点で旧版は失われています。JSON decoder が失敗しても、元へ戻す材料がありません。

```text
open with truncate -> write prefix -> process stops
                           |
                           v
                 target = incomplete JSON
```

守りたい invariant を先に書きます。

> target path には、最後に commit 済みの完全な document だけが見える。

これは「更新が必ず成功する」保証ではありません。失敗時に旧版を保つことで、失敗を retry・報告できる状態に留める保証です。

## prepare と commit を分ける

実行可能な最小例は [`examples/filereplace`](../../examples/filereplace/replace.go) にあります。

```go
if err := filereplace.Replace("tasks.json", encoded, 0o600); err != nil {
    return fmt.Errorf("save tasks: %w", err)
}
```

内部の順序は次のとおりです。

| 段階 | 操作 | target の状態 | この順序の理由 |
|---|---|---|---|
| 0 | memory 上で JSON を encode / validate | 旧版 | encode 失敗では filesystem に触れない |
| 1 | target と同じ directory に temp file を作る | 旧版 | rename が別 filesystem をまたがないようにする |
| 2 | temp に全 bytes を書く | 旧版 | target を truncate しない |
| 3 | temp を `Sync` して `Close` する | 旧版 | data error を commit 前に検出する |
| 4 | temp を target へ `Rename` する | 新版 | ここを一つの commit point にする |
| 5 | 必要なら parent directory を sync する | 新版 | directory entry の durability を要求する場合 |

temp file を system の temp directory に作ってはいけません。target と mount が違えば rename は失敗し、copy へ自動 fallback すると途中版が見える protocol に戻ります。

### `defer` は commit ではなく cleanup

例では rename 前の error で temp file を閉じて削除します。ただし process が `os.Exit`、kill、電源断で止まれば `defer` は走りません。したがって orphan temp file は起こり得ます。起動時 cleanup を加えるなら、prefix、所有者、十分に古いことを確認し、現在進行中の writer の file を消さない条件が必要です。

## process interruption を test する

「error を返す fake」だけでは、`defer` が実行されない停止を再現できません。例の test は子 process を二つの境界で終了させ、再起動する側から target を読みます。

```console
go test -run TestReplaceProcessInterruption -v ./examples/filereplace
go test -count=100 ./examples/filereplace
```

| 停止位置 | 再起動後に期待する target | temp artifact |
|---|---|---|
| temp の `Sync` 後、rename 前 | 旧版 | 残り得る |
| rename 後 | 新版 | rename 済みなので残らない |

この test が証明するのは、例の操作順序と process interruption の境界です。突然の電源断、filesystem firmware、network filesystem の cache までは process test だけで証明できません。

## OS と filesystem で保証が変わる

Go の [`os.Rename`](https://pkg.go.dev/os#Rename) は、non-Unix では同じ directory 内でも atomic operation ではないと明記しています。POSIX `rename()` は既存名が操作中も旧版か新版を指すことを規定しますが、directory 操作の atomicity と storage への durability は別です。

| 実行環境 | この例から主張できること | 追加判断 |
|---|---|---|
| Unix + rename を atomic に提供する local filesystem | rename 中の reader は旧版か新版を見る | power-loss 後の新版保持が必要なら parent directory の `fsync` も検討 |
| Windows / その他 non-Unix | temp への準備と target 直接 truncate の回避 | atomic replace が要件なら `ReplaceFile` 等の OS 固有実装と実機 test が必要 |
| NFS 等の network / distributed filesystem | 標準 API を呼んだことだけ | server、client cache、mount option、障害時 semantics を対象環境で検証 |

「Linux で test が通った」ことを、すべての volume の crash consistency へ一般化しません。container の bind mount、overlay filesystem、network volume も production と同じ構成で確認します。

## 失敗を commit point の前後で分類する

- **rename 前の error**: target は旧版のまま。temp を cleanup し、同じ desired state を retry できる。
- **rename 成功後の error**: target はすでに新版かもしれない。単純に「失敗したから未反映」と扱わず、read-back、generation、checksum で確認する。
- **二つの writer**: 各 rename が完全でも last-writer-wins で更新を失う。version / compare-and-swap、file lock、single writer のどれで直列化するか決める。

atomic replacement は transaction log ではありません。複数 file の一括更新、外部 API 呼び出しとの一括 commit、履歴保存が必要なら database、WAL、content-addressed generation + manifest など別の設計を選びます。

## 実務での判断基準

この pattern が合う例:

- 小さな設定 snapshot や単一 JSON document を全置換する。
- reader が file descriptor を開くたびに最新版を読み、旧版か新版のどちらでも処理できる。
- 単一 writer、または別途 concurrency control がある。

別方式を選ぶ例:

- 巨大 file を毎回全 encode するため、memory / I/O cost が要件を超える。
- append 履歴、複数 record の query、複数 writer が必要で database が適切。
- permission、ACL、owner、extended attribute を厳密に継承する必要がある。
- symlink をたどる path や攻撃者が書ける directory を扱い、path race への防御が必要。

## 演習

1. `Replace` の temp file を `os.TempDir()` に移し、なぜ同一 filesystem の前提が失われるか説明する。copy fallback は加えない。
2. `temporary.Sync()` を外した版を作り、process test が通っても power-loss durability の証拠にはならない理由を書く。
3. JSON encode を temp 作成後へ移して意図的に失敗させ、filesystem artifact と target の状態を比較する。
4. Task document に `generation` を追加し、二つの writer が同じ generation を更新しようとしたとき一方を拒否する契約を設計する。
5. orphan temp cleanup の条件を table-driven test にし、新しい temp file と別 user の file を削除しないことを示す。

この章の出口は、成功例だけでなく「どの停止位置で何が残るか」「どの環境では atomic と言えないか」を test 結果とともに説明できることです。

---

前: [package と API 設計](04-packages-api.md) | [セクション概要](README.md) | 次: [Go エンジニアリング](../02-engineering/README.md)
