package endpointsliceconditions

import "testing"

func TestDeriveAllInputs(t *testing.T) {
	tests := []struct {
		name            string
		podReady        bool
		terminating     bool
		publishNotReady bool
		want            Conditions
	}{
		{"not ready, active, default service", false, false, false, Conditions{}},
		{"not ready, active, publish not ready", false, false, true, Conditions{Ready: true}},
		{"not ready, terminating, default service", false, true, false, Conditions{Terminating: true}},
		{"not ready, terminating, publish not ready", false, true, true, Conditions{Ready: true, Terminating: true}},
		{"ready, active, default service", true, false, false, Conditions{Ready: true, Serving: true}},
		{"ready, active, publish not ready", true, false, true, Conditions{Ready: true, Serving: true}},
		{"ready, terminating, default service", true, true, false, Conditions{Serving: true, Terminating: true}},
		{"ready, terminating, publish not ready", true, true, true, Conditions{Ready: true, Serving: true, Terminating: true}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := Derive(test.podReady, test.terminating, test.publishNotReady)
			if got != test.want {
				t.Fatalf("Derive(%t, %t, %t) = %+v, want %+v", test.podReady, test.terminating, test.publishNotReady, got, test.want)
			}
			if test.terminating && !test.publishNotReady && got.Ready {
				t.Fatal("ordinary traffic must not see a terminating endpoint as ready")
			}
		})
	}
}
