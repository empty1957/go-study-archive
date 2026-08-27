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
2. [曖昧な結果と retry 契約](02-distributed-systems.md#前提-timeout-後の結果は二値ではない)で、冪等性、deadline、負荷予算を一つの判断にする。
3. [SLI/SLO](03-observability-sre.md#sli--slo)と [retry の増幅率](03-observability-sre.md#retry-を観測する)で選択の結果を観測する。
4. [security](04-security.md)と [compatibility](05-api-release.md)を release gate にする。
5. 最後に [product maturity](06-cncf-product-maturity.md)を技術・adoption・community の証拠で評価する。

各章は「前提 → failure model → 判断基準 → 実験 → 観測」の順で読みます。最初の実験として、Task API の readiness、routing propagation、graceful shutdown、強制終了を一つの timeline にしてください。次に [retry planner](02-distributed-systems.md#実行例-retry-planner)で response loss を再現し、可用性のための再送が重複 side effect や overload を増幅しない条件を test します。
