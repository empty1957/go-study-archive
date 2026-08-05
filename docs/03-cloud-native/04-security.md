# セキュリティと supply chain

## threat model を先に書く

asset、trust boundary、attacker capability、entry point、abuse case、mitigation、residual risk を列挙します。外部 API だけでなく、build system、plugin、configuration、operator 権限、backup も対象です。

## application security

- input size、型、範囲を境界で検証する。
- authentication と authorization を分け、resource ごとに認可する。
- least privilege の service account と filesystem permission。
- secret を log / metric / crash dump / image に入れない。
- TLS の certificate 検証を安易に無効化しない。
- archive 展開、path、URL fetch では traversal / SSRF を考慮する。
- resource exhaustion に rate / concurrency / memory limit を置く。

## supply chain

- dependency と toolchain を更新し、脆弱性情報を追う。
- release artifact を再現可能な CI で作る。
- source commit、build identity、artifact digest を結ぶ provenance。
- SBOM を生成し、artifact を署名・検証する。
- branch protection、review、短命 credential、MFA。
- maintainer 退任時に access を revoke する。

## vulnerability response

公開 security policy に private report 方法、対応 version、expected response、disclosure 方針を記載します。severity と exploitability を評価し、patch、release、advisory、downstream coordination を行います。security fix の diff が disclosure 前に漏れない process も必要です。

## Go 固有の確認

```console
go version -m ./your-binary # build/module 情報
go mod graph               # module graph
go list -m -u all          # 更新候補（自動更新前に release note を読む）
```

標準 Go vulnerability tooling（`govulncheck`）は別途 tool install が必要です。CI では version を pin し、検出結果を「到達可能性・runtime 条件・mitigation」とともに triage します。

