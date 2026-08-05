# CNCF Graduated への道

## まず誤解を解く

CNCF maturity は founder が申請書を埋めれば達成できる checklist ではありません。graduation は、価値ある cloud native problem を解き、広い adoption と独立した community を築き、安全で予測可能に運用されてきた結果です。要件は将来変わり得るため、申請時は必ず CNCF の最新公式 criteria を確認してください。

## 4 本の柱

### 1. Product / technology

- 狭く明確な problem statement と non-goals。
- 安定した API、upgrade、backup/restore、failure semantics。
- performance と scalability の再現可能な証拠。
- observability と day-2 operations。
- extension point は最小で versioned。

### 2. Security / release

- threat model、security policy、private disclosure。
- independent audit と指摘の remediation。
- signed artifact、SBOM、provenance、dependency process。
- predictable release / patch / EOL policy。

### 3. Adoption / ecosystem

- production user が複数組織に広がる。
- public adopters、case study、integration ecosystem。
- vendor-neutral branding と roadmap。
- migration と interoperability が実証される。

### 4. Community / governance

- governance、role、promotion、Code of Conduct、conflict process。
- maintainer が複数組織に分散。
- proposal と decision が公開される。
- contributor funnel、review latency、maintainer succession。
- 一社撤退でも release / security response を継続できる。

## 段階別の成果物

| 段階 | 技術 | Community / adoption |
|---|---|---|
| 0: prototype | 1 use case、local demo、設計記録 | 10 user interview、problem validation |
| 1: public alpha | install、docs、telemetry、security policy | public roadmap、issue template、初 contributor |
| 2: production beta | upgrade、backup、SLO、failure test | 3+ design partner、公開 case study |
| 3: stable | compatibility、audit、supply chain、release automation | 複数組織 maintainer、governance、定期 release |
| 4: ecosystem | conformance / plugin contract、scale evidence | 多様な adopter / integration / vendor |
| 5: CNCF path | criteria gap analysis、neutral infrastructure | Sandbox/Incubating/Graduated の最新 process に沿う |

## 半年ごとの scorecard

数値は vanity metric ではなく、risk を発見するために使います。

- active contributor / reviewer / maintainer の組織分布。
- issue first response、PR review、release cadence。
- production adopter と upgrade 成功事例。
- open security finding、patch lead time、dependency age。
- API deprecation、failed upgrade、rollback の件数。
- SLO、incident、postmortem action の完了率。
- bus factor と各 critical role の successor。

## 最初の 90 日

1. 解く問題を 1 文、non-goal を 5 個書く。
2. 20 人に既存 workflow と痛みを聞き、solution を売り込まない。
3. 1 user journey を end-to-end で動かす。
4. public repository に architecture、roadmap、license、contributing、security policy を置く。
5. telemetry と feedback から継続利用を確認する。
6. 独立した 2 人目が deploy / debug / contribute できるまで docs を直す。

