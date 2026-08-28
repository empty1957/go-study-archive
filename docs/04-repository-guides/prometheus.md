# Prometheus の読み方: scrape を証拠連鎖で診断する

公式 repository: [prometheus/prometheus](https://github.com/prometheus/prometheus)

この章では repository 全体を順番に読むのではなく、利用者から見える「series が更新されない」を入口にします。scrape の失敗、stale marker、query の lookback、label cardinality を一つの ingestion 契約として追い、source、test、live query がそれぞれ何を証明するかを分けます。

## 読解対象を固定する

| 項目 | 固定値 |
|---|---|
| release | [`v3.14.0`](https://github.com/prometheus/prometheus/releases/tag/v3.14.0) |
| commit | [`d7598b7141418fa35be2b5ec5d0fefb634199610`](https://github.com/prometheus/prometheus/tree/d7598b7141418fa35be2b5ec5d0fefb634199610) |
| 確認日 | 2026-08-28 |
| 主な範囲 | `scrape/`、`config/`、`tsdb/`、`docs/querying/` |

main branch の行番号や挙動は変わります。別版を読むときは tag と commit SHA を置き換え、同じ test が存在するか確認してください。Prometheus server 自体は安定した再利用 library を意図していないため、この章でも internal package を教材側から import しません。

## 前提と到達点

先に [可観測性と SRE](../03-cloud-native/03-observability-sre.md) の metric / SLI と、[性能設計](../02-engineering/05-performance.md) の「予算を先に決める」を確認します。この章を終えたら次を説明・実演できる状態を目指します。

- `up == 0` と series の不在を同じ意味として扱わない。
- scrape response の取得、parse、relabel、limit、commit、query を別の failure boundary として診断する。
- stale marker が「値 0」ではなく、以後の query から series を外す内部 sample だと説明する。
- label の直積から series 上限を事前計算し、scrape 単位と TSDB 全体の両方へ予算を置く。
- source、upstream test、縮小モデル、live query の証拠を混同しない。

## 利用者から見える invariant

この章では次を仮の運用契約にします。

> dashboard に現れる application series は、成功して commit された scrape の証拠である。scrape 自体の成否は `up` で別に観測し、消えた series は stale として query から除外する。新しい metric は per-target sample budget と fleet-wide active-series budget の両方を超えない。

`orders_total` の最後の値が画面に残っていても、現在の exporter が健康とは限りません。反対に、series が query 結果から消えても値が 0 になったとは限りません。この二つを分離することが出発点です。

## ingestion path と owner

```text
service discovery / config reload
  -> target と scrape loop の owner を作る
  -> HTTP response を timeout / body_size_limit 内で読む
  -> exposition data を parse する
  -> target label と metric_relabel_configs を適用する
  -> sample / label limit を検査する
  -> appender transaction を commit する
  -> TSDB Head / WAL / block に保持する
  -> PromQL が evaluation time と lookback / stale marker から選ぶ
```

主な地図は次のとおりです。

- `cmd/prometheus`: process の組み立てと flag。
- `config`: global / scrape config と limit。
- `discovery`: target group の発見。
- `scrape/manager.go`: scrape pool と loop の lifecycle。
- `scrape/scrape.go`: HTTP scrape、parse、staleness、report series、commit / rollback。
- `storage`, `tsdb`: appender、Head、WAL、block、compaction。
- `promql`, `web/api`: query evaluation と HTTP API。

call graph だけでは不十分です。scrape loop は target ごとに前回見えた series を cache し、現在の scrape と差分を取ります。したがって「HTTP request が返った」から「query に series が見える」までには、transaction と時系列上の状態があります。

## 読解 1: scrape の成功と失敗を分類する

固定版の [`scrapeAndReport`](https://github.com/prometheus/prometheus/blob/d7598b7141418fa35be2b5ec5d0fefb634199610/scrape/scrape.go#L1335-L1455) は、取得と append を行い、最後に report series を同じ appender へ追加して commit します。report series の定義と `up` の決定は [`report`](https://github.com/prometheus/prometheus/blob/d7598b7141418fa35be2b5ec5d0fefb634199610/scrape/scrape.go#L2027-L2174) で確認できます。

| 観測した事象 | transaction の扱い | 主な観測 | 読み違えやすい点 |
|---|---|---|---|
| 正常な response と append | application sample と report series を commit | `up=1`、sample 数、duration | `up=1` は application の意味的正しさまでは証明しない |
| network / timeout / response body 取得失敗 | empty scrape として前回 series を stale にし、report series を commit | `up=0`、body limit counter など | application metric の欠落だけでは failure 種別を特定できない |
| exposition の parse 失敗 | partial append を rollback し、empty scrape を再 append | `up=0`、target error | parse 前半だけが残るとは考えない |
| `sample_limit` / label limit 超過 | scrape 全体を失敗扱いにし、partial data を commit しない | `up=0`、limit counter | limit は一部を間引く throttle ではない |
| 成功した scrape から既存 series だけ消えた | その series へ stale marker を append | `up=1` でも対象 series は不在になる | exporter failure と metric の廃止を分ける |
| discovery から target が消えた | 再生成の競合を避けて約 2 scrape interval と 10% 待って stale 化 | 最終的に `up` 自体も不在 | `up==0` だけの alert は target 消失を拾わない |

重要な境界は三つあります。

1. `up` は exporter が公開する metric ではなく、Prometheus が scrape ごとに作る report series です。
2. 取得・parse失敗は empty scrape として stale 更新へ進みます。stale marker の追加は [`updateStaleMarkers`](https://github.com/prometheus/prometheus/blob/d7598b7141418fa35be2b5ec5d0fefb634199610/scrape/scrape.go#L1563-L1580) にあります。
3. `sample_limit` 超過は設定値を越えた sample だけ捨てるのではなく、scrape 全体を reject します。固定版の test は、失敗回に application sample が残らず、次の成功回で回復することを確認しています。

source を読んだ後、固定版 checkout で対象 test を実行します。

```console
git clone --depth 1 --branch v3.14.0 https://github.com/prometheus/prometheus.git
cd prometheus
go test ./scrape -run 'TestScrapeLoopRunCreatesStaleMarkersOn(FailedScrape|ParseFailure|SampleLimit)$' -count=1
```

対象は [`scrape_test.go`](https://github.com/prometheus/prometheus/blob/d7598b7141418fa35be2b5ec5d0fefb634199610/scrape/scrape_test.go#L2669-L2860) です。この test が証明するのは同一 process 内の scrape loop / appender の挙動です。実 network、service discovery の遅延、production の retention や alert routing は証明しません。

## 読解 2: stale と query 上の不在

stale marker は通常の `NaN` 表示や値 0 の代わりではありません。Prometheus が series の終了を内部表現し、marker より後の evaluation time でその series を返さないために使います。固定版の [query documentation](https://github.com/prometheus/prometheus/blob/d7598b7141418fa35be2b5ec5d0fefb634199610/docs/querying/basics.md#L469-L499) と source を対にして読みます。

- scrape に timestamp を明示しない通常の sample は、前回存在して今回消えれば stale marker が入る。
- target 自体の削除は、すぐ戻る target と競合しないよう遅延して stale 化する。
- Prometheus server の shutdown では、再起動後の継続を想定して end-of-run stale marker を書かない。
- exporter が明示 timestamp を付ける series は既定では同じ stale tracking を通らず、既定 5 分の lookback 後に消える。必要性を確認したうえで `track_timestamps_staleness` を検討する。
- stale marker 後に新しい sample が入れば、その series は再び query に現れる。

このため、次の二つは別の alert 条件です。

```promql
# 発見済み target の scrape が期間内に一度でも失敗した
min_over_time(up{job="api"}[5m]) == 0

# target/report series 自体が期間を通して存在しない
absent_over_time(up{job="api"}[5m])
```

`up == 0` だけでは discovery から消えた target を検出できず、`absent_over_time` だけでは発見済み target の連続失敗を詳しく示せません。実際の alert では、期待する target 数、maintenance、`for`、routing を加えて flapping と誤検知を制御します。

## 読解 3: cardinality を二つの予算へ接続する

1 series の identity は metric name と label set です。独立した label 候補数が `route=20`、`method=4`、`code_class=6` なら、1 target あたりの上限は `20 × 4 × 6 = 480` series です。50 target なら最大 24,000 series が加わります。実データでは値が相関して直積未満になることがありますが、設計 review では安全側の上限として使えます。

境界ごとの limit は同じ問題を解いていません。

| 制御 | 守る境界 | 保証しないこと |
|---|---|---|
| bounded label design | series 生成前 | target 数増加、series churn、実装ミス |
| `body_size_limit` | uncompressed response の読み取り量 | parse 後の series 数を正確には表さない |
| `metric_relabel_configs` | 不要 sample を ingestion 前に drop | network 転送と response parse の cost は残る |
| `sample_limit` | metric relabel 後の per-scrape sample 数 | fleet 全体の active series 上限、graceful degradation |
| `label_limit` / name / value length | 1 sample の label 数と byte 長 | label value の種類数、つまり cardinality |
| TSDB Head budget | process 全体の active series | どの team / metric が増加を所有するか |

`sample_limit` の既定 `0` は無制限です。超過時は scrape 全体が失敗するため、事故が起きてから値を下げるだけでは観測対象を丸ごと失う可能性があります。先に bounded label を設計し、canary target で実測し、limit は最後の防壁として置きます。設定の正確な意味は固定版の [`configuration.md`](https://github.com/prometheus/prometheus/blob/d7598b7141418fa35be2b5ec5d0fefb634199610/docs/configuration/configuration.md#L533-L573) で確認します。

この repository の縮小モデルは Prometheus internals を再実装せず、label 直積を review 時の判定へ変換します。

```console
go test -v ./examples/seriesbudget
```

[`examples/seriesbudget`](../../examples/seriesbudget/budget.go) は、per-target sample budget と fleet-wide Head series budget の両方を満たすときだけ `accept` します。`user_id` のように有限集合を説明できない label は reject します。相関、relabel、churn はモデル外なので、accept は production の安全を証明せず、live observation へ進める条件にすぎません。

## live observation の最小セット

staging の Prometheus と、応答を「正常」「metric を1つ省略」「500」「limit超過」へ切り替えられる fixture exporter を用意します。実験前に scrape interval、timeout、limit、対象 binary の version、開始時刻を記録します。

### 1. scrape 自体を観測する

```promql
up{job="fixture"}
scrape_duration_seconds{job="fixture"}
scrape_samples_scraped{job="fixture"}
scrape_samples_post_metric_relabeling{job="fixture"}
```

`extra_scrape_metrics: true` を設定した場合は、上限への接近も確認できます。

```promql
scrape_samples_post_metric_relabeling{job="fixture"}
  / (scrape_sample_limit{job="fixture"} > 0)
```

body / sample limit の発火は Prometheus 自身の `/metrics` から確認します。

```promql
increase(prometheus_target_scrapes_exceeded_body_size_limit_total[5m]) > 0
increase(prometheus_target_scrapes_exceeded_sample_limit_total[5m]) > 0
```

### 2. stale の時系列を記録する

1. 正常応答を2回成功させ、fixture metric と `up=1` を記録する。
2. HTTP 200のまま fixture metric だけ省略し、次の evaluation で series が消えることを確認する。
3. metric を戻し、再び現れることを確認する。
4. 500またはtimeoutを注入し、`up=0` と application series の不在を同じ時刻軸で記録する。
5. target を discovery から削除し、`up=0` ではなく `up` 自体が不在になるまでの時間を測る。

値を手で推測せず、query URLまたはAPI response、UTC timestamp、Prometheus log を evidence packet に残します。

### 3. series 数と churn を観測する

```promql
prometheus_tsdb_head_series
rate(prometheus_tsdb_head_series_created_total[15m])
rate(prometheus_tsdb_head_series_removed_total[15m])
```

Head series の現在値が横ばいでも、created と removed がともに高ければ label value が入れ替わる churn を疑います。固定版の metric 定義は [`tsdb/head.go`](https://github.com/prometheus/prometheus/blob/d7598b7141418fa35be2b5ec5d0fefb634199610/tsdb/head.go#L456-L500) で確認できます。所有者を絞るには Status > TSDB Status と `promtool tsdb analyze <data-dir>` を使い、調査対象 data directory の copy / access 権限を先に決めます。

## evidence ladder

| 証拠 | 証明できること | 証明できないこと |
|---|---|---|
| 固定版 source | branch と error path、commit / rollback の順序 | build した binary がその commit か |
| upstream unit test | 特定入力での staleness / limit invariant | 実 network、discovery、負荷下の timing |
| `seriesbudget` test | review policy の算術と境界条件 | Prometheus の内部挙動、実 cardinality |
| config と binary version | 実行時に意図した limit を渡したこと | reload が成功し全 target に反映されたこと |
| live PromQL / log | その時刻・環境の観測結果 | 別 cluster、将来の traffic、原因そのもの |
| profile / TSDB analysis | resource cost と高 cardinality の所在 | label を削る product 上の妥当性 |

「source に書いてある」で運用結果を断定せず、「dashboard で消えた」で原因を断定しません。隣り合う段を接続して仮説を狭めます。

## 失敗例と trade-off

- `label_limit` を設定したので cardinality は安全、と考える。これは1 sampleのlabel数を制限するだけで、`user_id` の種類数は制限しない。
- `sample_limit` を超えたら上限までの sample は残る、と考える。実際は scrape 全体が失敗し、`up=0` になる。
- incident 中に `sample_limit` だけ引き上げる。観測は戻っても、memory / WAL / query cost の事故を後段へ移す可能性がある。
- `up == 0` だけを alert にする。target が discovery から消えて `up` が stale になったケースを見逃す。
- `count(metric)` の現在値だけを見る。短命な label value が高速に入れ替わる churn を見逃す。
- `metric_relabel_configs` で高 cardinality metric を drop する。緊急防御にはなるが、転送・parse cost は残り、失った診断軸も戻らない。
- route の実 path、user ID、request ID、error message を label にする。有限な route template やerror classへ正規化し、個別値はsampled log / traceへ送る。

## 演習の完了条件

- [ ] `v3.14.0` の3つの scrape test を実行し、各 test が証明しない範囲も書いた。
- [ ] 正常、metric省略、HTTP失敗、sample limit超過、target削除の timeline を `up` と application metricで比較した。
- [ ] 新metricのlabel直積を `seriesbudget` に入力し、target数を10倍にした結果を説明した。
- [ ] `prometheus_tsdb_head_series` と created / removed rate を同時に記録し、増加とchurnを区別した。
- [ ] limitを上げる、labelを集約する、metricをdropする各案について、失う診断能力と守るresourceを書いた。
- [ ] source link、test command、config、query result、timestampを1つの読解ノートにまとめた。

次は同じ evidence chain を、query timeout / sample limit、または Head → WAL → block compaction の durability 境界へ延ばします。
