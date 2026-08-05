# アーキテクチャと依存関係

## 最初は modular monolith

network boundary は latency、partial failure、version skew、認証、観測、deployment coordination を生みます。team と scaling の必要がない段階で microservice に分割せず、process 内の明確な package boundary から始めます。

```text
cmd/taskapi
  └── transport/http ──> task.Service ──> task.Store
                                       └─ MemoryStore / SQLStore
```

依存矢印は domain 側へ向け、domain は HTTP や database driver を知りません。

## controller pattern

cloud native control plane では宣言的 reconciliation が重要です。

```text
observe current state
        ↓
compare with desired state
        ↓
perform one idempotent action
        ↓
record status / requeue with backoff
```

reconcile は何度呼んでも収束し、途中 crash 後も再開できるようにします。長い transaction にせず、status と event を利用者が診断できる形で残します。

## API evolution

- additive change を優先する。
- field の absent と zero を区別する必要を考える。
- consumer が未知 field / enum を扱えるか決める。
- deprecation window と migration guide を公開する。
- storage schema、wire API、CLI の compatibility を別々に定義する。
- rolling upgrade 中の version skew を test する。

## configuration

flag、file、environment、dynamic config の優先順位を明示します。secret を通常設定に混ぜず、起動時に検証し、effective config を secret を除いて診断可能にします。変更可能な config は atomic な snapshot として適用し、半分だけ反映しないようにします。

## failure を設計に入れる

architecture decision record (ADR) には happy path だけでなく、次を記録します。

- dependency が遅い / 応答しない / 間違った response を返す。
- process が書き込みの各段階で kill される。
- disk full、permission error、clock skew が起きる。
- old / new version が混在する。
- credential が漏洩・失効・rotation される。

## ADR テンプレート

```markdown
# ADR-NNN: 決定タイトル
Status: Proposed / Accepted / Superseded

## Context（制約と問題）
## Decision（決めたこと）
## Alternatives（比較した選択肢）
## Consequences（得るもの・失うもの）
## Failure modes and rollback
## Validation（どう正しさを確かめるか）
```

