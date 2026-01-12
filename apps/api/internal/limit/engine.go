package limit

import (
	"context"
	"sort"

	"golang.org/x/sync/errgroup"
)

// Engine orchestrates parallel limit checking.
// All registered checkers run concurrently via errgroup.
type Engine struct {
	checkers []Checker
}

// NewEngine creates a new limit engine with the given checkers.
// Checkers are run in parallel, order doesn't matter.
func NewEngine(checkers ...Checker) *Engine {
	return &Engine{checkers: checkers}
}

// Check runs all limit checks in parallel and returns the merged result.
// If any check fails, the highest priority failure reason is returned.
// Errors from checkers are treated as soft failures (logged but allowed).
func (e *Engine) Check(ctx context.Context, userID string) (*Result, error) {
	if len(e.checkers) == 0 {
		return &Result{Allowed: true}, nil
	}

	g, ctx := errgroup.WithContext(ctx)
	results := make([]*CheckResult, len(e.checkers))

	// Run all checks in parallel
	for i, checker := range e.checkers {
		i, checker := i, checker // Capture for goroutine
		g.Go(func() error {
			r, err := checker.Check(ctx, userID)
			if err != nil {
				// Log error but treat as allowed (fail open for availability)
				// TODO: Add structured logging
				results[i] = Allowed()
				return nil
			}
			results[i] = r
			return nil
		})
	}

	// Wait for all checks to complete
	if err := g.Wait(); err != nil {
		return nil, err
	}

	return e.merge(results), nil
}

// merge combines all check results into a single Result.
// The highest priority (lowest number) failure wins if multiple fail.
func (e *Engine) merge(results []*CheckResult) *Result {
	merged := &Result{
		Allowed:          true,
		RemainingDaily:   -1, // Default: unlimited
		RemainingMonthly: -1, // Default: unlimited
		Meta:             make(map[string]string),
	}

	// Collect failures and sort by priority
	var failures []*CheckResult
	for _, r := range results {
		if r == nil {
			continue
		}
		if !r.Allowed {
			failures = append(failures, r)
		}
		// Merge metadata from all results
		for k, v := range r.Meta {
			merged.Meta[k] = v
		}
	}

	// If no failures, request is allowed
	if len(failures) == 0 {
		return merged
	}

	// Sort failures by priority (lowest first)
	sort.Slice(failures, func(i, j int) bool {
		return failures[i].Priority < failures[j].Priority
	})

	// Use highest priority failure
	merged.Allowed = false
	merged.Reason = failures[0].Reason

	return merged
}

// AddChecker adds a checker to the engine.
// Can be used for dynamic checker registration.
func (e *Engine) AddChecker(checker Checker) {
	e.checkers = append(e.checkers, checker)
}

// CheckerCount returns the number of registered checkers.
func (e *Engine) CheckerCount() int {
	return len(e.checkers)
}
