package rolloutgate

import (
	"math"
	"testing"
	"time"
)

func TestEvaluate(t *testing.T) {
	policy := Policy{
		MinRequests:   100,
		MaxErrorRate:  0.01,
		MaxP95Latency: 250 * time.Millisecond,
	}
	ready := Preconditions{
		ArtifactPinned:     true,
		BackwardCompatible: true,
		RollbackRehearsed:  true,
	}

	tests := []struct {
		name          string
		preconditions Preconditions
		observation   Observation
		want          Decision
		wantReasons   int
	}{
		{
			name:          "healthy candidate is promoted",
			preconditions: ready,
			observation:   Observation{Requests: 100, Failures: 1, P95Latency: 250 * time.Millisecond},
			want:          Promote,
			wantReasons:   1,
		},
		{
			name:          "small healthy sample is held",
			preconditions: ready,
			observation:   Observation{Requests: 99, P95Latency: 100 * time.Millisecond},
			want:          Hold,
			wantReasons:   1,
		},
		{
			name:          "missing preconditions are all reported",
			preconditions: Preconditions{},
			observation:   Observation{Requests: 100, P95Latency: 100 * time.Millisecond},
			want:          Hold,
			wantReasons:   3,
		},
		{
			name:          "error budget breach rolls back",
			preconditions: ready,
			observation:   Observation{Requests: 100, Failures: 2, P95Latency: 100 * time.Millisecond},
			want:          Rollback,
			wantReasons:   1,
		},
		{
			name:          "latency budget breach rolls back",
			preconditions: ready,
			observation:   Observation{Requests: 100, P95Latency: 251 * time.Millisecond},
			want:          Rollback,
			wantReasons:   1,
		},
		{
			name:          "multiple budget breaches are preserved",
			preconditions: ready,
			observation:   Observation{Requests: 100, Failures: 2, P95Latency: 251 * time.Millisecond},
			want:          Rollback,
			wantReasons:   2,
		},
		{
			name:          "invariant overrides insufficient sample",
			preconditions: ready,
			observation:   Observation{Requests: 1, InvariantViolations: 1},
			want:          Rollback,
			wantReasons:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Evaluate(policy, tt.preconditions, tt.observation)
			if err != nil {
				t.Fatalf("Evaluate() error = %v", err)
			}
			if got.Decision != tt.want {
				t.Errorf("Evaluate() decision = %q, want %q; reasons: %v", got.Decision, tt.want, got.Reasons)
			}
			if len(got.Reasons) != tt.wantReasons {
				t.Errorf("Evaluate() returned %d reasons, want %d: %v", len(got.Reasons), tt.wantReasons, got.Reasons)
			}
		})
	}
}

func TestEvaluateRejectsInvalidEvidence(t *testing.T) {
	validPolicy := Policy{MinRequests: 1, MaxErrorRate: 0.1, MaxP95Latency: time.Second}
	tests := []struct {
		name        string
		policy      Policy
		observation Observation
	}{
		{name: "zero sample target", policy: Policy{MaxErrorRate: 0.1, MaxP95Latency: time.Second}},
		{name: "NaN error budget", policy: Policy{MinRequests: 1, MaxErrorRate: math.NaN(), MaxP95Latency: time.Second}},
		{name: "failures exceed requests", policy: validPolicy, observation: Observation{Requests: 1, Failures: 2}},
		{name: "negative latency", policy: validPolicy, observation: Observation{P95Latency: -time.Millisecond}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Evaluate(tt.policy, Preconditions{}, tt.observation); err == nil {
				t.Fatal("Evaluate() error = nil, want invalid evidence error")
			}
		})
	}
}
