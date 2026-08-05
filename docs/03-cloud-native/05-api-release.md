# API、互換性、リリース

## 公開契約を棚卸しする

契約は Go API だけではありません。

- HTTP/gRPC、event schema、CRD。
- CLI flag、exit code、stdout format。
- configuration file / environment variable。
- metric name / label、log event。
- persisted data、backup format。
- extension / plugin interface。
- support する Go / OS / architecture / Kubernetes version。

それぞれ compatibility policy と deprecation period を定めます。

## release train

1. scope freeze と変更分類。
2. test、fuzz、security scan、license check。
3. upgrade / downgrade / version-skew test。
4. signed tag と reproducible build pipeline。
5. artifact、checksum、SBOM、provenance、release notes。
6. canary / staged rollout と health criteria。
7. rollback または roll-forward の演習。
8. support branch と EOL の公開。

release を特定個人の laptop から作らず、review 可能な automation にします。

## SemVer の注意

SemVer は API の定義があって初めて意味を持ちます。`v1.2.3` の数字だけで CLI、config、storage の互換性は決まりません。Go module の major version import path と、product 全体の release version が異なる場合もあります。

## migration の原則

- expand-and-contract: 先に新旧両対応、移行後に旧形式を削除。
- reader を先に、writer を後に更新。
- irreversible migration 前に backup と restore を検証。
- rolling upgrade 中に old/new が混在しても安全。
- deprecation warning は利用者が action を取れる情報を含む。

