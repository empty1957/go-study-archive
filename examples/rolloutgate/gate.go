// Package rolloutgate turns release evidence into an explicit rollout decision.
package rolloutgate

import (
	"errors"
	"fmt"
	"math"
	"time"
)

// Decision is the next allowed rollout action.
type Decision string

const (
	Promote  Decision = "promote"
	Hold     Decision = "hold"
	Rollback Decision = "rollback"
)

// Policy defines the evidence needed to promote a release.
type Policy struct {
	MinRequests   int
	MaxErrorRate  float64
	MaxP95Latency time.Duration
}

// Preconditions are facts that must be recorded before exposing a release.
type Preconditions struct {
	ArtifactPinned     bool
	BackwardCompatible bool
	RollbackRehearsed  bool
}

// Observation is one evaluation window for the candidate release.
type Observation struct {
	Requests            int
	Failures            int
	P95Latency          time.Duration
	InvariantViolations int
}

// Result contains a machine-readable action and the evidence behind it.
type Result struct {
	Decision Decision
	Reasons  []string
}

// Evaluate applies fail-closed rollout rules in safety order.
//
// An invariant violation rolls back immediately. Missing preconditions or an
// undersized sample hold the rollout. Only a sufficiently observed candidate
// inside both budgets may be promoted.
func Evaluate(policy Policy, preconditions Preconditions, observation Observation) (Result, error) {
	if err := validate(policy, observation); err != nil {
		return Result{}, err
	}

	if observation.InvariantViolations > 0 {
		return Result{
			Decision: Rollback,
			Reasons:  []string{"safety invariant violation observed"},
		}, nil
	}

	var missing []string
	if !preconditions.ArtifactPinned {
		missing = append(missing, "release artifact is not pinned by immutable identity")
	}
	if !preconditions.BackwardCompatible {
		missing = append(missing, "schema or API is not backward compatible")
	}
	if !preconditions.RollbackRehearsed {
		missing = append(missing, "rollback has not been rehearsed")
	}
	if len(missing) > 0 {
		return Result{Decision: Hold, Reasons: missing}, nil
	}

	if observation.Requests < policy.MinRequests {
		return Result{
			Decision: Hold,
			Reasons: []string{fmt.Sprintf(
				"insufficient sample: got %d requests, need %d",
				observation.Requests,
				policy.MinRequests,
			)},
		}, nil
	}

	errorRate := float64(observation.Failures) / float64(observation.Requests)
	var breaches []string
	if errorRate > policy.MaxErrorRate {
		breaches = append(breaches, fmt.Sprintf(
			"error rate %.4f exceeds %.4f",
			errorRate,
			policy.MaxErrorRate,
		))
	}
	if observation.P95Latency > policy.MaxP95Latency {
		breaches = append(breaches, fmt.Sprintf(
			"p95 latency %s exceeds %s",
			observation.P95Latency,
			policy.MaxP95Latency,
		))
	}
	if len(breaches) > 0 {
		return Result{Decision: Rollback, Reasons: breaches}, nil
	}

	return Result{
		Decision: Promote,
		Reasons:  []string{"sample is sufficient and all release budgets are satisfied"},
	}, nil
}

func validate(policy Policy, observation Observation) error {
	if policy.MinRequests <= 0 {
		return errors.New("minimum requests must be positive")
	}
	if math.IsNaN(policy.MaxErrorRate) || policy.MaxErrorRate < 0 || policy.MaxErrorRate > 1 {
		return errors.New("maximum error rate must be between 0 and 1")
	}
	if policy.MaxP95Latency <= 0 {
		return errors.New("maximum p95 latency must be positive")
	}
	if observation.Requests < 0 || observation.Failures < 0 || observation.Failures > observation.Requests {
		return errors.New("request and failure counts are inconsistent")
	}
	if observation.P95Latency < 0 {
		return errors.New("p95 latency must not be negative")
	}
	if observation.InvariantViolations < 0 {
		return errors.New("invariant violations must not be negative")
	}
	return nil
}
