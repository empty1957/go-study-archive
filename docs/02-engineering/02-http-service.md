# HTTP service の境界契約

HTTP handler は JSON を domain method へ渡すだけの接着剤ではありません。信頼できない入力が CPU、memory、goroutine、下流 connection を消費する入口です。この章では Task API を使い、request を受ける前に次の 2 つの予算を固定します。

- **入力予算**: 1 request が読める body は最大 16 KiB。
- **capacity 予算**: 1 process が同時に処理へ入れる API request は既定で 64 件。

[Go の基礎](../01-foundations/README.md)にある error 分類と package 境界、[並行処理](01-concurrency-context.md)にある owner / cancel / join が前提です。この章の問いは「処理能力を超えた仕事を、resource を消費する前にどの契約で拒否するか」です。

## request path と予算を使う位置

```text
socket / Server timeout
  -> routing
  -> admission: process 内の同時処理枠を取る
       ├─ 満杯: 503 + overloaded + Retry-After（handler は開始しない）
       └─ 取得: defer で枠を返す
            -> media type / body byte limit / JSON decode
            -> domain validation
            -> service -> store
            -> encode
```

server timeout は遅い connection が時間を無制限に使うのを防ぎ、body limit は 1 request の空間使用量を抑え、admission control は process 全体の同時使用量を抑えます。どれか 1 つで他を代替できません。たとえば 16 KiB の小さい request でも、下流 I/O が停止した 1 万件を同時に受ければ goroutine と connection は増え続けます。

本教材では [`internal/task/http.go`](../../internal/task/http.go) が `/v1/` の前に admission を置きます。`/healthz` と `/readyz` は枠の外に置き、過負荷中も観測できるようにします。

## 入力予算: 読む前から境界を作る

`POST /v1/tasks` は次の順で入力を分類します。

1. `Content-Type` を `mime.ParseMediaType` で解析し、`application/json` 以外と未指定を `415 Unsupported Media Type` にする。
2. 既知の `Content-Length` が 16 KiB を超えれば、body を decode せず `413 Content Too Large` にする。
3. `http.MaxBytesReader` で streaming / chunked body にも実際の read 上限を課す。
4. 未知 field、複数 JSON value、構文 error を `400 Bad Request` にする。
5. transport として正しい JSON の title rule 違反を domain validation の `invalid_title` にする。

| response | error code | caller が直すもの | 同じ内容の再試行 |
|---|---|---|---|
| `400` | `invalid_json` / `invalid_title` | JSON または domain input | 修正なしでは無意味 |
| `413` | `request_too_large` | body size | 小さくするまで無意味 |
| `415` | `unsupported_media_type` | `Content-Type` /表現形式 | header または形式を直す |
| `503` | `overloaded` | server capacity は一時的 | delay と budget を付けて候補 |

`Content-Length` だけは信用しません。header がない転送や chunked body があるため、read する reader 自体を制限します。逆に `MaxBytesReader` だけに任せると、サイズが既知でも上限を超えるまで読み始めます。2 段の検査は早い拒否と実測の強制を分担します。

圧縮 request を受ける API では「圧縮前 byte」と「展開後 byte」の両方が予算です。本教材は request compression を契約に含めていません。proxy が body を展開する構成では、application が受け取る時点のサイズと proxy 自身の上限を別々に確認します。

## capacity 予算: queue を作る前に拒否する

Task API の admission control は capacity 付き channel を semaphore として使い、空きがなければ待たずに拒否します。

```go
select {
case inFlight <- struct{}{}:
    defer func() { <-inFlight }()
    next.ServeHTTP(w, r)
default:
    w.Header().Set("Retry-After", "1")
    writeError(w, http.StatusServiceUnavailable, "overloaded", "...")
}
```

待機 queue を追加すると短い burst を吸収できますが、queue 中の request も connection、memory、deadline を消費します。下流が回復しなければ待ち時間が増え、caller timeout と server 側の処理開始が競合して「caller は諦めたが server は後から実行した」という曖昧さを増やします。学習用 API はこの trade-off を見えるように、**queue 長 0、即時拒否**を選びます。

既定の 64 は正解値ではなく安全装置の初期値です。`TASKAPI_MAX_IN_FLIGHT` で正の整数へ変更できます。値は次の順で決めます。

1. request 種別、下流 connection 数、CPU / memory limit を含む workload を固定する。
2. concurrency を増やし、throughput と p95/p99 latency、error、allocation、下流 saturation を同時に測る。
3. throughput が頭打ちになる手前に上限を置き、burst と health endpoint の余白を残す。
4. `in_flight`、reject 数、latency を観測し、replica 数や下流 capacity と一緒に再評価する。

Little's Law の直感 `concurrency ≈ throughput × latency` は出発点ですが、burst や heavy/light request の混在を表しません。すべてを同じ 1 枠として数える方式は単純な一方、重い request を区別できないという限界があります。[性能と profiling](05-performance.md#capacity-は拒否を含めて検証する)で測定条件を作ります。

## health endpoint

health endpoint は負荷を受ける application path と同じ予算で塞がないようにし、liveness と readiness の意味も個別 request の拒否から分けます。

### `503`、`429`、readiness を混同しない

- `503 Service Unavailable`: process 全体が一時的に処理能力を失った、またはこの admission 枠が満杯。本教材はこちら。
- `429 Too Many Requests`: tenant、credential、client など主体別の rate / quota policy に使う。認証前後のどこで数えるかも契約になる。
- readiness failure: instance を新規 routing 先から外す signal。瞬間的に 64 枠が埋まるたび readiness を落とすと、残りの instance へ負荷を寄せて悪化させ得る。

そのため現在の Task API は一時的な枠不足でも `/readyz` を成功させ、個別 request を `503` にします。長時間回復しない下流障害や process 自体が安全に処理できない状態では、別の状態機械として readiness を落とします。停止時の readiness、routing propagation、drain は [Pod の終了契約](../03-cloud-native/01-containers-kubernetes.md#pod-削除で並行して起きること)へ続きます。

## retry は response code だけでは決まらない

`Retry-After: 1` は caller に最低 1 秒待つ候補を伝えます。1 秒後の成功を予約するものではありません。全 caller が同時に戻る thundering herd を避けるため、client は jitter、試行回数または総経過時間の上限、caller の deadline を組み合わせます。

この実装の `overloaded` response は service method を呼ぶ**前**に生成されるため、その response を受け取った caller は同じ POST を再送できます。ただし、request が受理された後に response だけ失われた場合は区別できません。create 全般を安全に retry 可能にするには idempotency key と保存済み結果の再送が別途必要です。proxy が生成した別の `503` を application の `overloaded` と同一視しないことも重要です。

process local の 64 枠は cluster 全体の上限でも fairness policy でもありません。replica が N 個なら概算上は N 倍の request が下流へ到達し得ます。下流 connection pool、gateway の rate limit、tenant quota と予算を重ねる場合、各層が返す code と観測点を決めます。

## 失敗例から設計を選ぶ

| 失敗例 | 起きること | 判断 |
|---|---|---|
| goroutine を無制限に開始 | 下流停止時に memory と tail latency が増え続ける | 処理開始前に同時数を制限 |
| admission 枠が空くまで無期限に待つ | request deadline と queue 待ちが隠れ、timeout 後に処理し得る | queue 長と待機期限を明示。本教材は即時拒否 |
| `Content-Length` だけ検査 | streaming body が上限を迂回する | reader にも上限を課す |
| decode error をすべて `400` にする | caller が縮小すべきか構文修正すべきか分からない | `MaxBytesError` を `413` に分類 |
| overload ごとに readiness failure | routing 先が減って残存 replica の負荷が増す | 個別拒否と instance 退避を別状態にする |
| すべての `503` POST を自動再試行 | response loss 後に重複作成し得る | 未開始を示す application code か idempotency 契約を使う |

## 実行可能な契約テスト

```console
go test -run='TestHandlerRejects(OversizedBody|OverloadBeforeStartingWork)$' -v ./internal/task
go test -count=100 -run='TestHandlerRejectsOverloadBeforeStartingWork$' ./internal/task
go test -race ./internal/task
```

[`TestHandlerRejectsOversizedBody`](../../internal/task/http_test.go) は既知の長さと `ContentLength = -1` の streaming 相当を同じ `413 / request_too_large` 契約で検証します。

[`TestHandlerRejectsOverloadBeforeStartingWork`](../../internal/task/http_test.go) は `time.Sleep` で負荷を推測せず、blocking store の event を使います。

```text
request A が admission 枠を取得
  -> store.Create 到達を channel で観測し停止
  -> request B は 503 / overloaded（store へ到達しない）
  -> /healthz は 200
  -> A を解放して枠を返す
  -> request C は 200
```

演習:

1. `TASKAPI_MAX_IN_FLIGHT=1 go run ./cmd/taskapi` で起動し、遅い store を注入して拒否を観測する。
2. admission を「最大 2 件待つ queue」へ変更し、queue 待ち時間と caller cancel を event で test する。
3. `overloaded` の client を作り、jitter 付き retry と総 retry budget を table test にする。
4. POST の response loss を再現し、idempotency key なしで重複作成が起きることを示す。
5. [技能チェックリスト](../checklist.md#service--production)へ設定値、負荷条件、reject 率、判断を証拠として残す。

## server timeout と response の残課題

`http.Server` の `ReadHeaderTimeout`、`ReadTimeout`、`WriteTimeout`、`IdleTimeout` も workload に合わせます。streaming endpoint では一律の `WriteTimeout` が不適切な場合があります。下流 call には request context と個別 timeout を渡します。

response は status code を body より先に書き、内部 error や secret を返さず、安定した JSON error schema を保ちます。request ID / trace ID、認証・認可、pagination、per-tenant rate limit、metrics/tracing、OpenAPI、distributed idempotency は意図的な次の課題です。middleware の順序を変えると、認証されない大量 body を読むか、拒否 request をどの主体へ計上できるかも変わります。変更時は [テスト戦略](03-testing.md#resource-境界は-event-で固定する)に沿って failure path を先に固定してください。
