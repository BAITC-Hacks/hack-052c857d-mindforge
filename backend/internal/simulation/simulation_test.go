package simulation

import "testing"

func TestExampleScenario(t *testing.T) {
	r, err := Simulate([]Choice{{"M7", "nura"}, {"M8", "nura"}, {"M10", "nura"}, {"M12", ""}, {"M5", "saryarka"}})
	if err != nil {
		t.Fatal(err)
	}
	if r.Spent != 95 || r.Score < 56 || r.Score > 57 {
		t.Fatalf("got spent=%d score=%.2f", r.Spent, r.Score)
	}
}
func TestRejectsConflict(t *testing.T) {
	_, err := Simulate([]Choice{{"M1", "esil"}, {"M3", "nura"}, {"M4", "almaty"}, {"M10", "nura"}, {"M12", ""}})
	if err == nil {
		t.Fatal("expected conflict")
	}
}
