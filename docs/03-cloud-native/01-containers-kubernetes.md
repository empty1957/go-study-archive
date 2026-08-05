# コンテナと Kubernetes

## container の境界

container image は root filesystem と実行設定の配布形式で、実行時の隔離は主に OS の namespace、cgroup、capability 等が担います。container は VM と同じ security boundary だと無条件に仮定しません。

Go binary を image にする際の要点:

- multi-stage build と小さな runtime image。
- non-root user、read-only root filesystem、不要 capability の削除。
- architecture / OS を明確にし、CGO の有無を理解。
- digest pinning、SBOM、署名、provenance。
- signal を受け取る PID 1 と graceful shutdown。
- writable path、CA certificate、timezone data の必要性。

## Kubernetes の control loop

利用者は desired state を API Server に宣言し、controller が observed state を望ましい状態へ収束させます。scheduler は未配置 Pod の node を選び、kubelet は node 上で Pod lifecycle を実現します。永続 state の中核には etcd が使われます。

```text
kubectl/client -> API Server -> etcd
                       ↑
             controllers / scheduler
                       ↓
                    kubelet -> container runtime
```

## application 側の責任

Kubernetes は application を自動的に reliable にはしません。

- termination signal を処理し、猶予内に drain する。
- readiness を rollout と dependency 状態に対応させる。
- resource request / limit を計測に基づき設定する。
- replica 間で local disk / memory state を共有できると仮定しない。
- disruption、reschedule、duplicate execution に耐える。
- Pod IP や起動順序を永続 identity として扱わない。

## operator を作る前に

CRD/controller は declarative domain に有効ですが、versioned API と長期保守の責任を生みます。Helm chart や通常の service API で十分か、reconciliation が user value を生むか、削除/finalizer/upgrade を安全に扱えるかを先に検討します。

