// Package retrycontract turns retry safety, load, and time constraints into
// an explicit decision. It deliberately leaves HTTP and sleeping to callers.
package retrycontract

import (
	"errors"
	"math"
	"time"
)

// Action is the next action allowed by the retry contract.
type Action string

const (
	Stop  Action = "stop"
	Retry Action = "retry"
)

// FailureKind is a dependency-specific classification, not a guess based on
// every 4xx or 5xx response.
type FailureKind string

const (
	Permanent  FailureKind = "permanent"
	Transient  FailureKind = "transient"
	Overloaded FailureKind = "overloaded"
	Ambiguous  FailureKind = "ambiguous"
)

// Policy bounds one logical operation, including its initial attempt.
type Policy struct {
	MaxAttempts    int
	TotalTimeout   time.Duration
	AttemptTimeout time.Duration
	MinAttemptTime time.Duration
	BaseBackoff    time.Duration
	MaxBackoff     time.Duration
}

// State is the evidence available after a failed attempt.
type State struct {
	CompletedAttempts    int
	Elapsed              time.Duration
	RetryBudgetAvailable bool
}

// Failure describes why the last attempt failed. RetryAfter is a lower bound
// supplied by the dependency contract, such as an accepted Retry-After value.
type Failure struct {
	Kind       FailureKind
	RetryAfter time.Duration
}

// Decision contains the schedule for a next attempt or the reason to stop.
type Decision struct {
	Action         Action
	Delay          time.Duration
	AttemptTimeout time.Duration
	Reason         string
}

// Decide applies safety gates before scheduling a retry.
//
// replaySafe must be established from operation semantics: for example an
// idempotent method, durable idempotency-key deduplication, or proof that the
// previous attempt was not applied. jitter must be in [0, 1] and is injected
// so callers can use randomness in production and exact values in tests.
func Decide(policy Policy, replaySafe bool, failure Failure, state State, jitter float64) (Decision, error) {
	if err := validate(policy, failure, state, jitter); err != nil {
		return Decision{}, err
	}

	if failure.Kind == Permanent {
		return stop("failure is permanent"), nil
	}
	if !replaySafe {
		return stop("operation is not proven safe to replay"), nil
	}
	if !state.RetryBudgetAvailable {
		return stop("shared retry budget is exhausted"), nil
	}
	if state.CompletedAttempts >= policy.MaxAttempts {
		return stop("maximum attempts reached"), nil
	}

	delayCap := exponentialCap(policy, state.CompletedAttempts)
	delay := time.Duration(float64(delayCap) * jitter)
	if failure.RetryAfter > delay {
		delay = failure.RetryAfter
	}

	remaining := policy.TotalTimeout - state.Elapsed
	if remaining <= delay {
		return stop("total deadline leaves no time for another attempt"), nil
	}

	available := remaining - delay
	if available < policy.MinAttemptTime {
		return stop("total deadline leaves too little time for a useful attempt"), nil
	}
	attemptTimeout := policy.AttemptTimeout
	if available < attemptTimeout {
		attemptTimeout = available
	}

	return Decision{
		Action:         Retry,
		Delay:          delay,
		AttemptTimeout: attemptTimeout,
		Reason:         "retry is replay-safe and inside attempt, time, and load budgets",
	}, nil
}

func exponentialCap(policy Policy, completedAttempts int) time.Duration {
	cap := policy.BaseBackoff
	for retryNumber := 1; retryNumber < completedAttempts; retryNumber++ {
		if cap >= policy.MaxBackoff || cap > policy.MaxBackoff/2 {
			return policy.MaxBackoff
		}
		cap *= 2
	}
	if cap > policy.MaxBackoff {
		return policy.MaxBackoff
	}
	return cap
}

func stop(reason string) Decision {
	return Decision{Action: Stop, Reason: reason}
}

func validate(policy Policy, failure Failure, state State, jitter float64) error {
	if policy.MaxAttempts < 1 {
		return errors.New("maximum attempts must be positive")
	}
	if policy.TotalTimeout <= 0 || policy.AttemptTimeout <= 0 || policy.MinAttemptTime <= 0 {
		return errors.New("total, maximum attempt, and minimum attempt timeouts must be positive")
	}
	if policy.MinAttemptTime > policy.AttemptTimeout {
		return errors.New("minimum attempt time must not exceed attempt timeout")
	}
	if policy.MinAttemptTime > policy.TotalTimeout {
		return errors.New("minimum attempt time must not exceed total timeout")
	}
	if policy.BaseBackoff <= 0 || policy.MaxBackoff < policy.BaseBackoff {
		return errors.New("backoff bounds are inconsistent")
	}
	if state.CompletedAttempts < 1 || state.Elapsed < 0 {
		return errors.New("retry state is inconsistent")
	}
	if failure.RetryAfter < 0 {
		return errors.New("retry-after must not be negative")
	}
	switch failure.Kind {
	case Permanent, Transient, Overloaded, Ambiguous:
	default:
		return errors.New("failure kind is unknown")
	}
	if math.IsNaN(jitter) || jitter < 0 || jitter > 1 {
		return errors.New("jitter must be between zero and one")
	}
	return nil
}
