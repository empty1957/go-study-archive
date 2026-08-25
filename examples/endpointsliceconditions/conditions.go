// Package endpointsliceconditions contains a dependency-free learning model of
// the Kubernetes EndpointSlice condition projection. It is an executable note,
// not a replacement for Kubernetes API types.
package endpointsliceconditions

// Conditions is the observable EndpointSlice state derived from a Pod and its
// owning Service.
type Conditions struct {
	Ready       bool
	Serving     bool
	Terminating bool
}

// Derive projects Pod readiness and deletion state into endpoint conditions.
//
// Ready preserves compatibility with consumers that must not route ordinary
// traffic to terminating endpoints. PublishNotReadyAddresses is the explicit
// Service-level exception to that rule.
func Derive(podReady, terminating, publishNotReadyAddresses bool) Conditions {
	serving := podReady
	ready := publishNotReadyAddresses || (serving && !terminating)

	return Conditions{
		Ready:       ready,
		Serving:     serving,
		Terminating: terminating,
	}
}
