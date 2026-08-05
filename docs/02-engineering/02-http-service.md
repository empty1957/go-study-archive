# HTTP service の設計

## request path

```text
socket -> Server timeout -> routing -> middleware -> handler
       -> decode/validate -> service -> store -> encode
```

各層が何を保証し、どの error をどこへ変換するかを決めます。本教材では標準 `http.ServeMux` を使い、router library の知識より HTTP の契約を見えるようにしています。

## server timeout

`http.Server` の `ReadHeaderTimeout`、`ReadTimeout`、`WriteTimeout`、`IdleTimeout` を workload に合わせて設定します。streaming endpoint では一律の `WriteTimeout` が不適切な場合があります。下流 call にも request context と個別 timeout を渡します。

## input の扱い

- request body の最大サイズを制限する。
- `Content-Type` を検証する。
- JSON の未知 field を拒否するか互換性方針として決める。
- parse error と domain validation error を分ける。
- client に内部実装や secret を含む error を返さない。
- collection には page size の上限を置く。

## response の契約

- status code を body より先に書く。
- JSON error schema を安定させる。
- create は `201 Created` と `Location` が有用。
- request ID / trace ID を error response と log で対応させる。
- retry 可能なら `Retry-After` や documented error code を検討する。

## middleware

middleware は横断的関心事に限定します。request ID、authentication、access log、panic recovery、metrics、tracing などです。domain rule を middleware に隠しません。

順序には意味があります。panic recovery が最外周なら内側の panic を捕捉できます。認証前に body を大量に読むと DoS 面が増えます。metric label に raw path や user ID を入れると cardinality が爆発します。

## health endpoint

- `/healthz`（liveness）: process が回復不能に停止していないか。依存 DB の一時停止だけで process を再起動し続けない。
- `/readyz`（readiness）: 現在 traffic を安全に処理できるか。起動中、drain 中、必須依存が使えない場合は false。
- startup probe: 初期化が長い application の起動猶予。

## production への追加課題

本教材の API は学習用で、次は意図的に省いています: 認証・認可、永続 DB、rate limit、TLS、metrics/tracing、pagination、request ID、OpenAPI、distributed deployment。これらを 1 つずつ追加し、各変更に failure test を付けるのが Phase 2〜3 の演習です。

