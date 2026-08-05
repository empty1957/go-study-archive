# Prometheus の読み方

公式 repository: [prometheus/prometheus](https://github.com/prometheus/prometheus)

Prometheus は monitoring system と time-series database を 1 つの server に持ち、pull model、multi-dimensional data model、PromQL を特徴とします。公式 README は、この repository 自体を再利用 library として安定化しているわけではない点も明記しています。library 利用なら `prometheus/common` や `client_golang` など用途別 repository を検討します。

## 何を学ぶか

- command で多数の subsystem を組み立てる方法。
- service discovery → scrape → append → TSDB の ingestion path。
- PromQL parse / engine / storage query の境界。
- WAL、block、compaction、retention。
- high-cardinality data と resource control。

## 地図

- `cmd/prometheus`: server entry point。
- `config`: configuration model / load。
- `discovery`: target discovery。
- `scrape`: scrape loop と ingestion。
- `storage`, `tsdb`: storage abstraction と local TSDB。
- `promql`: parser と query evaluation。
- `rules`, `notifier`: rule evaluation と alert notification。
- `web`: HTTP/API/UI boundary。

現在の構成は [公式 repository](https://github.com/prometheus/prometheus) で確認してください。

## 読解 1: scrape sample の旅

1. config reload から target / scrape pool 作成を追う。
2. 1 回の HTTP scrape の timeout と body limit を探す。
3. exposition data の parse と label 処理を追う。
4. appender interface から TSDB head / WAL までを追う。
5. duplicate / out-of-order / stale sample の扱いを調べる。
6. 成功・失敗・duration を表す internal metric を探す。

## 読解 2: query

HTTP API から PromQL parse、engine、storage querier への call path を追います。lookback、step、timeout、sample limit が CPU / memory をどう保護するか、concurrent query と cancellation がどこで効くかを確認します。

## 読解 3: TSDB

head（変更中の最新 data）、WAL（recovery）、immutable block、compaction を区別します。process kill のどの地点でも acknowledged data の保証がどうなるか、checksum / repair、retention と mmap の resource behavior を test から読みます。

## 演習

- 1 metric の label cardinality が 10 倍になったとき、memory/disk/query に及ぶ影響を説明する。
- config reload が atomic か、subsystem 間の partial apply をどう扱うか追う。
- `/metrics` の self-observation から scrape failure を診断する最小 dashboard を作る。
- 公式 README の「stand-alone program で library ではない」という契約が package 利用判断にどう影響するか書く。

