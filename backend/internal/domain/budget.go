package domain

import (
	"errors"
	"fmt"
	"math"
)

var ErrBudgetExceeded = errors.New("selected initiatives exceed the simulation budget")

// BudgetSummary reports the selected initiatives' total cost and remaining budget.
type BudgetSummary struct {
	TotalCost       int64 `json:"total_cost"`
	RemainingBudget int64 `json:"remaining_budget"`
}

// ValidateBudget totals initiative costs and rejects selections that exceed budget.
func ValidateBudget(budget int64, selected []Initiative) (BudgetSummary, error) {
	if budget < 0 {
		return BudgetSummary{}, errors.New("simulation budget cannot be negative")
	}

	var total int64
	for _, initiative := range selected {
		if initiative.Cost < 0 {
			return BudgetSummary{}, fmt.Errorf("initiative %q has a negative cost", initiative.ID)
		}
		if initiative.Cost > math.MaxInt64-total {
			return BudgetSummary{}, errors.New("total initiative cost overflows int64")
		}
		total += initiative.Cost
	}
	if total > budget {
		return BudgetSummary{}, ErrBudgetExceeded
	}

	return BudgetSummary{TotalCost: total, RemainingBudget: budget - total}, nil
}
