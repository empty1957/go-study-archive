package seriesbudget

import (
	"math"
	"reflect"
	"testing"
)

func TestEvaluate(t *testing.T) {
	t.Parallel()

	base := Budget{
		Targets:                 50,
		CurrentSamplesPerTarget: 1_000,
		MaxSamplesPerTarget:     2_000,
		CurrentHeadSeries:       100_000,
		MaxHeadSeries:           150_000,
	}

	tests := []struct {
		name       string
		dimensions []Dimension
		budget     Budget
		want       Result
	}{
		{
			name: "bounded labels fit both budgets",
			dimensions: []Dimension{
				{Name: "route", Values: 20, Bounded: true},
				{Name: "method", Values: 4, Bounded: true},
				{Name: "code_class", Values: 6, Bounded: true},
			},
			budget: base,
			want: Result{
				Decision:                  Accept,
				SeriesPerTarget:           480,
				ProjectedSamplesPerTarget: 1_480,
				ProjectedHeadSeries:       124_000,
				Reasons:                   []string{"projection fits the scrape and active-series budgets"},
			},
		},
		{
			name: "unbounded user label is rejected",
			dimensions: []Dimension{
				{Name: "user_id", Values: 1, Bounded: false},
			},
			budget: base,
			want: Result{
				Decision: Reject,
				Reasons:  []string{"label \"user_id\" has no finite value bound"},
			},
		},
		{
			name: "both budgets are reported",
			dimensions: []Dimension{
				{Name: "route", Values: 100, Bounded: true},
				{Name: "tenant", Values: 20, Bounded: true},
			},
			budget: base,
			want: Result{
				Decision:                  Reject,
				SeriesPerTarget:           2_000,
				ProjectedSamplesPerTarget: 3_000,
				ProjectedHeadSeries:       200_000,
				Reasons: []string{
					"projected samples per target 3000 exceed budget 2000",
					"projected head series 200000 exceed budget 150000",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := Evaluate(tt.dimensions, tt.budget)
			if err != nil {
				t.Fatalf("Evaluate() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("Evaluate() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestEvaluateRejectsOverflow(t *testing.T) {
	t.Parallel()

	_, err := Evaluate(
		[]Dimension{{Name: "explosive", Values: math.MaxUint64, Bounded: true}},
		Budget{
			Targets:                 2,
			MaxSamplesPerTarget:     math.MaxUint64,
			CurrentSamplesPerTarget: 1,
			MaxHeadSeries:           math.MaxUint64,
		},
	)
	if err == nil {
		t.Fatal("Evaluate() error = nil, want overflow error")
	}
}
