package retrycontract

import (
	"math"
	"testing"
	"time"
)

func TestDecide(t *testing.T) {
	policy := Policy{
		MaxAttempts:    4,
		TotalTimeout:   5 * time.Second,
		AttemptTimeout: time.Second,
		MinAttemptTime: 100 * time.Millisecond,
		BaseBackoff:    100 * time.Millisecond,
		MaxBackoff:     800 * time.Millisecond,
	}
	ready := State{CompletedAttempts: 1, RetryBudgetAvailable: true}

	tests := []struct {
		name             string
		replaySafe       bool
		failure          Failure
		state            State
		jitter           float64
		wantAction       Action
		wantDelay        time.Duration
		wantAttemptLimit time.Duration
	}{
		{
			name:             "transient failure uses full jitter",
			replaySafe:       true,
			failure:          Failure{Kind: Transient},
			state:            ready,
			jitter:           0.5,
			wantAction:       Retry,
			wantDelay:        50 * time.Millisecond,
			wantAttemptLimit: time.Second,
		},
		{
			name:             "server delay is a floor",
			replaySafe:       true,
			failure:          Failure{Kind: Overloaded, RetryAfter: 2 * time.Second},
			state:            ready,
			jitter:           0.5,
			wantAction:       Retry,
			wantDelay:        2 * time.Second,
			wantAttemptLimit: time.Second,
		},
		{
			name:       "ambiguous outcome without replay proof stops",
			replaySafe: false,
			failure:    Failure{Kind: Ambiguous},
			state:      ready,
			jitter:     0.5,
			wantAction: Stop,
		},
		{
			name:             "idempotency contract permits ambiguous retry",
			replaySafe:       true,
			failure:          Failure{Kind: Ambiguous},
			state:            ready,
			jitter:           0.5,
			wantAction:       Retry,
			wantDelay:        50 * time.Millisecond,
			wantAttemptLimit: time.Second,
		},
		{
			name:       "permanent failure stops",
			replaySafe: true,
			failure:    Failure{Kind: Permanent},
			state:      ready,
			jitter:     0.5,
			wantAction: Stop,
		},
		{
			name:       "shared retry budget exhaustion stops",
			replaySafe: true,
			failure:    Failure{Kind: Transient},
			state:      State{CompletedAttempts: 1},
			jitter:     0.5,
			wantAction: Stop,
		},
		{
			name:       "attempt budget exhaustion stops",
			replaySafe: true,
			failure:    Failure{Kind: Transient},
			state:      State{CompletedAttempts: 4, RetryBudgetAvailable: true},
			jitter:     0.5,
			wantAction: Stop,
		},
		{
			name:       "retry-after beyond total deadline stops",
			replaySafe: true,
			failure:    Failure{Kind: Overloaded, RetryAfter: time.Second},
			state:      State{CompletedAttempts: 1, Elapsed: 4500 * time.Millisecond, RetryBudgetAvailable: true},
			jitter:     0,
			wantAction: Stop,
		},
		{
			name:             "last attempt is clipped to total deadline",
			replaySafe:       true,
			failure:          Failure{Kind: Transient},
			state:            State{CompletedAttempts: 1, Elapsed: 4500 * time.Millisecond, RetryBudgetAvailable: true},
			jitter:           0,
			wantAction:       Retry,
			wantAttemptLimit: 500 * time.Millisecond,
		},
		{
			name:       "too little useful attempt time stops",
			replaySafe: true,
			failure:    Failure{Kind: Transient},
			state:      State{CompletedAttempts: 1, Elapsed: 4950 * time.Millisecond, RetryBudgetAvailable: true},
			jitter:     0,
			wantAction: Stop,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Decide(policy, tt.replaySafe, tt.failure, tt.state, tt.jitter)
			if err != nil {
				t.Fatalf("Decide() error = %v", err)
			}
			if got.Action != tt.wantAction {
				t.Fatalf("Decide() action = %q, want %q; reason: %s", got.Action, tt.wantAction, got.Reason)
			}
			if got.Delay != tt.wantDelay {
				t.Errorf("Decide() delay = %s, want %s", got.Delay, tt.wantDelay)
			}
			if got.AttemptTimeout != tt.wantAttemptLimit {
				t.Errorf("Decide() attempt timeout = %s, want %s", got.AttemptTimeout, tt.wantAttemptLimit)
			}
			if got.Reason == "" {
				t.Error("Decide() reason is empty")
			}
		})
	}
}

func TestDecideCapsExponentialBackoff(t *testing.T) {
	policy := Policy{
		MaxAttempts:    20,
		TotalTimeout:   time.Minute,
		AttemptTimeout: time.Second,
		MinAttemptTime: 100 * time.Millisecond,
		BaseBackoff:    100 * time.Millisecond,
		MaxBackoff:     800 * time.Millisecond,
	}
	state := State{CompletedAttempts: 10, RetryBudgetAvailable: true}

	got, err := Decide(policy, true, Failure{Kind: Transient}, state, 1)
	if err != nil {
		t.Fatalf("Decide() error = %v", err)
	}
	if got.Delay != policy.MaxBackoff {
		t.Fatalf("Decide() delay = %s, want cap %s", got.Delay, policy.MaxBackoff)
	}
}

func TestDecideRejectsInvalidContract(t *testing.T) {
	validPolicy := Policy{
		MaxAttempts:    2,
		TotalTimeout:   time.Second,
		AttemptTimeout: 500 * time.Millisecond,
		MinAttemptTime: 50 * time.Millisecond,
		BaseBackoff:    10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
	}
	validFailure := Failure{Kind: Transient}
	validState := State{CompletedAttempts: 1, RetryBudgetAvailable: true}

	tests := []struct {
		name    string
		policy  Policy
		failure Failure
		state   State
		jitter  float64
	}{
		{name: "zero attempts", policy: Policy{TotalTimeout: time.Second, AttemptTimeout: time.Second, MinAttemptTime: time.Millisecond, BaseBackoff: time.Millisecond, MaxBackoff: time.Second}, failure: validFailure, state: validState, jitter: 0.5},
		{name: "minimum attempt exceeds maximum", policy: Policy{MaxAttempts: 2, TotalTimeout: time.Second, AttemptTimeout: time.Millisecond, MinAttemptTime: time.Second, BaseBackoff: time.Millisecond, MaxBackoff: time.Second}, failure: validFailure, state: validState, jitter: 0.5},
		{name: "minimum attempt exceeds total", policy: Policy{MaxAttempts: 2, TotalTimeout: time.Millisecond, AttemptTimeout: time.Second, MinAttemptTime: time.Second, BaseBackoff: time.Millisecond, MaxBackoff: time.Second}, failure: validFailure, state: validState, jitter: 0.5},
		{name: "backoff bounds reversed", policy: Policy{MaxAttempts: 2, TotalTimeout: time.Second, AttemptTimeout: time.Second, MinAttemptTime: time.Millisecond, BaseBackoff: time.Second, MaxBackoff: time.Millisecond}, failure: validFailure, state: validState, jitter: 0.5},
		{name: "unknown failure", policy: validPolicy, failure: Failure{}, state: validState, jitter: 0.5},
		{name: "negative elapsed", policy: validPolicy, failure: validFailure, state: State{CompletedAttempts: 1, Elapsed: -time.Millisecond}, jitter: 0.5},
		{name: "NaN jitter", policy: validPolicy, failure: validFailure, state: validState, jitter: math.NaN()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Decide(tt.policy, true, tt.failure, tt.state, tt.jitter); err == nil {
				t.Fatal("Decide() error = nil, want invalid contract error")
			}
		})
	}
}
