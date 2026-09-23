package simulation

import (
	"math"
	"testing"
)

func TestExampleScenario(t *testing.T) {
	r, err := Simulate([]Choice{{"M7", "nura"}, {"M8", "nura"}, {"M10", "nura"}, {"M12", ""}, {"M5", "saryarka"}})
	if err != nil {
		t.Fatal(err)
	}
	if r.Spent != 95 {
		t.Fatalf("spent=%d, want 95", r.Spent)
	}
	if math.Abs(r.BaselineScore-52.56) > 0.2 {
		t.Fatalf("baseline score=%.2f, want approx 52.56", r.BaselineScore)
	}
	if math.Abs(r.Score-56.5) > 0.5 {
		t.Fatalf("score=%.2f, want approx 56.5", r.Score)
	}
	if !hasSynergy(r, "M10", "M12") {
		t.Fatal("expected M10+M12 synergy to activate")
	}
}

func TestRejectsConflict(t *testing.T) {
	_, err := Simulate([]Choice{{"M1", "esil"}, {"M3", "nura"}, {"M4", "almaty"}, {"M10", "nura"}, {"M12", ""}})
	if err == nil {
		t.Fatal("expected conflict")
	}
}

func TestRejectsMissingDistrictForDistrictInitiative(t *testing.T) {
	_, err := Simulate([]Choice{{"M7", ""}, {"M8", "nura"}, {"M10", "nura"}, {"M12", ""}, {"M5", "saryarka"}})
	if err == nil {
		t.Fatal("expected missing district to be rejected")
	}
}

func TestCheapStrategyIsValid(t *testing.T) {
	r, err := Simulate([]Choice{{"M9", "esil"}, {"M11", "nura"}, {"M10", "saryarka"}, {"M12", ""}, {"M4", "almaty"}})
	if err != nil {
		t.Fatal(err)
	}
	if r.Spent != 61 {
		t.Fatalf("spent=%d, want 61", r.Spent)
	}
	if r.Score <= 0 {
		t.Fatalf("score=%.2f, want positive score", r.Score)
	}
}

func hasSynergy(r Result, a, b string) bool {
	for _, s := range r.Synergies {
		if hasString(s.Initiatives, a) && hasString(s.Initiatives, b) {
			return true
		}
	}
	return false
}

func hasString(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}
