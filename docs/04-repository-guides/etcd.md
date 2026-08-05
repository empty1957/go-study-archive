# etcd の読み方

公式 repository: [etcd-io/etcd](https://github.com/etcd-io/etcd)

etcd は critical な分散 system data のための key-value store で、gRPC API、TLS、Raft による replicated log を中核に持ちます。公式 repository は `api`、`client`、`raft`、`server` 等を versioned Go module として案内しています。

## 何を学ぶか

- consensus algorithm と product の責務の違い。
- proposal → replicated log → committed entry → state machine apply。
- WAL / snapshot / backend の durability と recovery。
- watch、lease、compaction、linearizable read。
- robustness / fault injection test。

## module の地図

main branch の正確な tree は変わるため、[公式 repository](https://github.com/etcd-io/etcd) と対象 tag の `go.work` / `go.mod` を確認します。読む単位は概ね:

- `api`: protobuf と public contract。
- `client/v3`: client、retry、balancing、watch / lease API。
- `raft`: algorithm の library。
- `server`: network、storage、Raft node、state machine integration。
- `etcdctl`: CLI client。
- tests / robustness tooling: failure と history 検証。

## 読解 1: Put の旅

1. gRPC KV API と request type を確認。
2. server handler から proposal を作る地点へ進む。
3. request ID と response 待機の対応を探す。
4. Raft の Ready/advance loop で message、WAL、committed entry の順序を追う。
5. state machine / backend への apply と revision 更新を追う。
6. client response が返る時点の durability / visibility guarantee を説明する。

## 読解 2: linearizable read

leader が最新 commit を認識している保証、quorum confirmation、read index、apply index の関係を追います。serializable read との latency / availability trade-off を client-visible semantics から説明します。

## 読解 3: watch と compaction

revision の列を watch consumer が追う経路、slow consumer、buffer、compacted revision、reconnect 時の再開を確認します。watch は単なる channel ではなく、履歴 retention と client recovery protocol を含むことが分かります。

## 演習

- 3 node cluster で 1 node / leader / network minority を止め、read/write 結果を予想してから実験する。
- `raft` package と `server` package の責務境界を表にする。
- acknowledged write の直後に kill した recovery path を test から追う。
- lease だけで外部 resource の排他的所有が安全かを考え、fencing token の必要性を書く。

