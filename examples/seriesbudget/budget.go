// Package seriesbudget estimates the upper bound of a proposed metric's
// series count before the metric reaches a Prometheus scrape boundary.
//
// It is a policy model, not a Prometheus implementation. Real label values can
// be correlated, and series can churn over time, so operators must compare the
// estimate with live TSDB metrics after rollout.
package seriesbudget

import (
	"errors"
	"fmt"
	"math"
)

// Decision is the next allowed action for a metric proposal.
type Decision string

const (
	Accept Decision = "accept"
	Reject Decision = "reject"
)

// Dimension describes one label on the proposed metric.
type Dimension struct {
	Name    string
	Values  uint64
	Bounded bool
}

// Budget makes both the per-target scrape budget and the fleet-wide active
// series budget explicit. MaxSamplesPerTarget corresponds to an operational
// policy that can be enforced with Prometheus's sample_limit.
type Budget struct {
	Targets                 uint64
	CurrentSamplesPerTarget uint64
	MaxSamplesPerTarget     uint64
	CurrentHeadSeries       uint64
	MaxHeadSeries           uint64
}

// Result records the projection and every violated budget.
type Result struct {
	Decision                  Decision
	SeriesPerTarget           uint64
	ProjectedSamplesPerTarget uint64
	ProjectedHeadSeries       uint64
	Reasons                   []string
}

// Evaluate multiplies label dimensions to obtain a conservative upper bound.
// A label whose value set is not bounded is rejected before arithmetic.
func Evaluate(dimensions []Dimension, budget Budget) (Result, error) {
	if err := validateBudget(budget); err != nil {
		return Result{}, err
	}

	seriesPerTarget := uint64(1)
	for _, dimension := range dimensions {
		if dimension.Name == "" {
			return Result{}, errors.New("label dimension name must not be empty")
		}
		if dimension.Values == 0 {
			return Result{}, fmt.Errorf("label dimension %q must have at least one value", dimension.Name)
		}
		if !dimension.Bounded {
			return Result{
				Decision: Reject,
				Reasons:  []string{fmt.Sprintf("label %q has no finite value bound", dimension.Name)},
			}, nil
		}

		var err error
		seriesPerTarget, err = multiply(seriesPerTarget, dimension.Values)
		if err != nil {
			return Result{}, fmt.Errorf("project label combinations: %w", err)
		}
	}

	projectedSamples, err := add(budget.CurrentSamplesPerTarget, seriesPerTarget)
	if err != nil {
		return Result{}, fmt.Errorf("project samples per target: %w", err)
	}
	newFleetSeries, err := multiply(seriesPerTarget, budget.Targets)
	if err != nil {
		return Result{}, fmt.Errorf("project fleet series: %w", err)
	}
	projectedHead, err := add(budget.CurrentHeadSeries, newFleetSeries)
	if err != nil {
		return Result{}, fmt.Errorf("project head series: %w", err)
	}

	result := Result{
		Decision:                  Accept,
		SeriesPerTarget:           seriesPerTarget,
		ProjectedSamplesPerTarget: projectedSamples,
		ProjectedHeadSeries:       projectedHead,
	}
	if projectedSamples > budget.MaxSamplesPerTarget {
		result.Decision = Reject
		result.Reasons = append(result.Reasons, fmt.Sprintf(
			"projected samples per target %d exceed budget %d",
			projectedSamples,
			budget.MaxSamplesPerTarget,
		))
	}
	if projectedHead > budget.MaxHeadSeries {
		result.Decision = Reject
		result.Reasons = append(result.Reasons, fmt.Sprintf(
			"projected head series %d exceed budget %d",
			projectedHead,
			budget.MaxHeadSeries,
		))
	}
	if result.Decision == Accept {
		result.Reasons = []string{"projection fits the scrape and active-series budgets"}
	}
	return result, nil
}

func validateBudget(budget Budget) error {
	if budget.Targets == 0 {
		return errors.New("target count must be positive")
	}
	if budget.MaxSamplesPerTarget == 0 {
		return errors.New("maximum samples per target must be positive")
	}
	if budget.MaxHeadSeries == 0 {
		return errors.New("maximum head series must be positive")
	}
	return nil
}

func multiply(a, b uint64) (uint64, error) {
	if a != 0 && b > math.MaxUint64/a {
		return 0, errors.New("uint64 overflow")
	}
	return a * b, nil
}

func add(a, b uint64) (uint64, error) {
	if b > math.MaxUint64-a {
		return 0, errors.New("uint64 overflow")
	}
	return a + b, nil
}
