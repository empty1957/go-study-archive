# containerd の読み方

公式 repository: [containerd/containerd](https://github.com/containerd/containerd)

containerd は host 上の container lifecycle、image transfer/storage、execution supervision 等を管理する daemon で、CNCF Graduated project です。直接 end-user が使うより大きな system に組み込まれることを意図しています（[公式 README](https://github.com/containerd/containerd)）。

## 何を学ぶか

- client と daemon の責務分離。
- gRPC API と internal implementation の境界。
- built-in / external plugin architecture。
- content-addressed storage、snapshotter、runtime shim。
- Linux / Windows と複数 runtime を扱う portability。

## 最初に読む文書

- [PLUGINS.md](https://github.com/containerd/containerd/blob/main/docs/PLUGINS.md): smart client model と plugin 種別。
- [runtime-v2.md](https://github.com/containerd/containerd/blob/main/docs/runtime-v2.md): daemon、shim、runtime の関係。
- [content-flow.md](https://github.com/containerd/containerd/blob/main/docs/content-flow.md): image content から実行までの流れ。

## 読解 1: client から container 作成

1. `cmd/ctr` の command から Go client call を探す。
2. client option が OCI spec、snapshot、runtime 選択をどう作るか追う。
3. protobuf API と daemon service implementation を対応させる。
4. metadata と content store の state transition を書く。
5. shim process が container lifecycle を daemon から分離する理由を説明する。
6. cancel / cleanup 中に残り得る resource と test を探す。

## 読解 2: plugin

公式 plugin 文書では、content store、snapshotter、diff service、runtime などの拡張点が説明されています。built-in implementation も plugin として扱うことで boundary を実戦で検証している点に注目します。

確認する問い:

- plugin identity と dependency はどう宣言されるか。
- init failure は daemon 起動全体を止めるか。
- proxy plugin の trust / availability boundary はどこか。
- interface の versioning と external process の protocol をどう安定させるか。

## 読解 3: content と snapshot

digest で識別する immutable content と、container root filesystem を提供する snapshot を区別します。download、unpack、lease、garbage collection の間で「参照中の content を消さない」invariant を追うと、storage lifecycle の設計教材になります。

## 演習

- `ctr plugins ls` の output と source の registration を対応付ける。
- fake content store / snapshotter を使う test を読み、interface boundary を図示する。
- runtime が crash、daemon が restart、disk が full の各場合に state がどう回収されるか仮説を立て、test/docs で検証する。

