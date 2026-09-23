package domain

import (
	"errors"
	"testing"
)

func TestValidateBudget(t *testing.T) {
	tests := []struct {
		name      string
		budget    int64
		selected  []Initiative
		wantTotal int64
		wantLeft  int64
		wantErr   error
	}{
		{
			name:   "valid selection",
			budget: 100,
			selected: []Initiative{
				{ID: "bus-lane", Cost: 30},
				{ID: "tree-planting", Cost: 20},
			},
			wantTotal: 50,
			wantLeft:  50,
		},
		{
			name:   "exact budget",
			budget: 50,
			selected: []Initiative{
				{ID: "clinic", Cost: 35},
				{ID: "lighting", Cost: 15},
			},
			wantTotal: 50,
			wantLeft:  0,
		},
		{
			name:   "over budget",
			budget: 49,
			selected: []Initiative{
				{ID: "clinic", Cost: 35},
				{ID: "lighting", Cost: 15},
			},
			wantErr: ErrBudgetExceeded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateBudget(tt.budget, tt.selected)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ValidateBudget() error = %v, want %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if got.TotalCost != tt.wantTotal || got.RemainingBudget != tt.wantLeft {
				t.Errorf("ValidateBudget() = %+v, want total %d and remaining %d", got, tt.wantTotal, tt.wantLeft)
			}
		})
	}
}
