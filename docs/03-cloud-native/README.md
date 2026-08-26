# 03: クラウドネイティブ

1. [コンテナと Kubernetes](01-containers-kubernetes.md)
2. [分散システム](02-distributed-systems.md)
3. [可観測性と SRE](03-observability-sre.md)
4. [セキュリティと supply chain](04-security.md)
5. [API、互換性、リリース](05-api-release.md)
6. [CNCF Graduated への道](06-cncf-product-maturity.md)

クラウドネイティブとは YAML を書くことではありません。障害・変更・成長を前提に、観測と自動化を備えた system を、開かれた ecosystem で持続的に運営することです。

## このセクションの進め方

先に [Go エンジニアリング](../02-engineering/README.md)の Task API、`context`、HTTP shutdown を実行してください。このセクションではそれらを一つの process から複数 replica と control loop の世界へ広げます。

1. [Pod の終了契約](01-containers-kubernetes.md#pod-削除で並行して起きること)で application と platform の責任境界を描く。
2. [部分障害](02-distributed-systems.md)を前提に retry、冪等性、一貫性を選ぶ。
3. [SLI/SLO](03-observability-sre.md#sli--slo)で選択の結果を観測する。
4. [security](04-security.md)と [compatibility](05-api-release.md)を release gate にする。
5. 最後に [product maturity](06-cncf-product-maturity.md)を技術・adoption・community の証拠で評価する。

各章は「前提 → failure model → 判断基準 → 実験 → 観測」の順で読みます。最初の実験として、Task API の readiness、routing propagation、graceful shutdown、強制終了を一つの timeline にしてください。
