// Package simulation contains the deterministic, auditable city model.
// It deliberately does not call an LLM: an LLM may explain this output, but
// all scores must stay reproducible.
package simulation

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
	"sort"
)

const Budget = 100

type Metrics struct {
	T1 float64 `json:"t1"`
	T2 float64 `json:"t2"`
	E1 float64 `json:"e1"`
	E2 float64 `json:"e2"`
	S1 float64 `json:"s1"`
	S2 float64 `json:"s2"`
	B1 float64 `json:"b1"`
	B2 float64 `json:"b2"`
	C1 float64 `json:"c1"`
	C2 float64 `json:"c2"`
}
type District struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Profile    string  `json:"profile"`
	Population float64 `json:"population"`
	Metrics    Metrics `json:"metrics"`
}
type Initiative struct {
	ID        string  `json:"id"`
	Direction string  `json:"direction"`
	Name      string  `json:"name"`
	Scope     string  `json:"scope"`
	Cost      int     `json:"cost"`
	Lag       int     `json:"lag"`
	Effects   Metrics `json:"effects"`
}
type Choice struct {
	InitiativeID string `json:"initiative_id"`
	DistrictID   string `json:"district_id,omitempty"`
}
type DistrictResult struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Before Metrics `json:"before"`
	After  Metrics `json:"after"`
	Score  float64 `json:"score"`
}
type Result struct {
	Budget          int              `json:"budget"`
	Spent           int              `json:"spent"`
	Remaining       int              `json:"remaining"`
	Score           float64          `json:"score"`
	BaselineScore   float64          `json:"baseline_score"`
	Delta           float64          `json:"delta"`
	CriticalCount   int              `json:"critical_count"`
	Districts       []DistrictResult `json:"districts"`
	Strengths       []string         `json:"strengths"`
	Risks           []string         `json:"risks"`
	Recommendations []string         `json:"recommendations"`
}

type dataset struct {
	Districts   []District   `json:"districts"`
	Initiatives []Initiative `json:"initiatives"`
}

//go:embed data/city.json
var rawDataset []byte
var Districts, Initiatives = loadDataset()

func loadDataset() ([]District, []Initiative) {
	var d dataset
	if err := json.Unmarshal(rawDataset, &d); err != nil {
		panic("invalid embedded city dataset: " + err.Error())
	}
	return d.Districts, d.Initiatives
}

func catalog() map[string]Initiative {
	r := map[string]Initiative{}
	for _, x := range Initiatives {
		r[x.ID] = x
	}
	return r
}
func findDistrict(id string) (int, bool) {
	for i, d := range Districts {
		if d.ID == id {
			return i, true
		}
	}
	return 0, false
}
func add(a *Metrics, b Metrics, k float64) {
	a.T1 += b.T1 * k
	a.T2 += b.T2 * k
	a.E1 += b.E1 * k
	a.E2 += b.E2 * k
	a.S1 += b.S1 * k
	a.S2 += b.S2 * k
	a.B1 += b.B1 * k
	a.B2 += b.B2 * k
	a.C1 += b.C1 * k
	a.C2 += b.C2 * k
}
func clip(a *Metrics) {
	p := []*float64{&a.T1, &a.T2, &a.E1, &a.E2, &a.S1, &a.S2, &a.B1, &a.B2, &a.C1, &a.C2}
	for _, v := range p {
		*v = math.Max(0, math.Min(100, *v))
	}
}
func districtScore(m Metrics) float64 {
	return .10*m.T1 + .10*m.T2 + .09*m.E1 + .11*m.E2 + .11*m.S1 + .11*m.S2 + .09*m.B1 + .09*m.B2 + .10*m.C1 + .10*m.C2
}
func critical(m Metrics) int {
	n := 0
	for _, v := range []float64{m.T1, m.T2, m.E1, m.E2, m.S1, m.S2, m.B1, m.B2, m.C1, m.C2} {
		if v < 40 {
			n++
		}
	}
	return n
}
func cityScore(ms []Metrics) (float64, int) {
	avg, min, crit := 0.0, 101.0, 0
	for i, m := range ms {
		s := districtScore(m)
		avg += Districts[i].Population * s
		if s < min {
			min = s
		}
		crit += critical(m)
	}
	return .7*avg + .3*min - float64(crit), crit
}

// Validate checks every business rule before calculations begin.
func Validate(choices []Choice) ([]Initiative, error) {
	if len(choices) != 5 {
		return nil, fmt.Errorf("нужно принять ровно 5 решений, выбрано: %d", len(choices))
	}
	cat := catalog()
	seen := map[string]bool{}
	dirs := map[string]int{}
	chosen := map[string]Choice{}
	total := 0
	for _, c := range choices {
		x, ok := cat[c.InitiativeID]
		if !ok {
			return nil, fmt.Errorf("неизвестное мероприятие %q", c.InitiativeID)
		}
		if seen[x.ID] {
			return nil, fmt.Errorf("мероприятие %s выбрано повторно", x.ID)
		}
		seen[x.ID] = true
		chosen[x.ID] = c
		dirs[x.Direction]++
		if dirs[x.Direction] > 2 {
			return nil, fmt.Errorf("в направлении %s нельзя выбрать более 2 мер", x.Direction)
		}
		total += x.Cost
		if x.Scope == "district" {
			if _, ok := findDistrict(c.DistrictID); !ok {
				return nil, fmt.Errorf("для %s укажите существующий район", x.ID)
			}
		} else if c.DistrictID != "" {
			return nil, fmt.Errorf("для городской меры %s район не указывается", x.ID)
		}
	}
	if total > Budget {
		return nil, fmt.Errorf("бюджет превышен: %d из %d", total, Budget)
	}
	if seen["M1"] && seen["M3"] {
		return nil, fmt.Errorf("M1 и M3 несовместимы")
	}
	for _, pair := range [][2]string{{"M4", "M7"}, {"M5", "M13"}} {
		if seen[pair[0]] && seen[pair[1]] && chosen[pair[0]].DistrictID == chosen[pair[1]].DistrictID {
			return nil, fmt.Errorf("%s и %s нельзя размещать в одном районе", pair[0], pair[1])
		}
	}
	items := make([]Initiative, 0, 5)
	for _, c := range choices {
		items = append(items, cat[c.InitiativeID])
	}
	return items, nil
}

func Simulate(choices []Choice) (Result, error) {
	items, err := Validate(choices)
	if err != nil {
		return Result{}, err
	}
	after := make([]Metrics, len(Districts))
	before := make([]Metrics, len(Districts))
	spent := 0
	for i, d := range Districts {
		after[i] = d.Metrics
		before[i] = d.Metrics
	}
	for i, x := range items {
		spent += x.Cost
		k := float64(8-x.Lag) / 8
		if x.Scope == "city" {
			for d := range after {
				add(&after[d], x.Effects, k)
			}
		} else {
			d, _ := findDistrict(choices[i].DistrictID)
			add(&after[d], x.Effects, k)
		}
	}
	// Fixed, non-lagged synergy bonuses target the district of the first district-level measure.
	lookup := map[string]Choice{}
	for _, c := range choices {
		lookup[c.InitiativeID] = c
	}
	for _, s := range []struct {
		a, b string
		e    Metrics
	}{{"M1", "M2", Metrics{T1: 2}}, {"M10", "M12", Metrics{B1: 2}}, {"M5", "M6", Metrics{E2: 2}}} {
		if _, ok := lookup[s.a]; ok {
			if _, ok2 := lookup[s.b]; ok2 {
				d, _ := findDistrict(lookup[s.a].DistrictID)
				add(&after[d], s.e, 1)
			}
		}
	}
	for i := range after {
		clip(&after[i])
	}
	score, crit := cityScore(after)
	base, _ := cityScore(before)
	r := Result{Budget: Budget, Spent: spent, Remaining: Budget - spent, Score: round(score), BaselineScore: round(base), Delta: round(score - base), CriticalCount: crit}
	for i, d := range Districts {
		r.Districts = append(r.Districts, DistrictResult{d.ID, d.Name, before[i], after[i], round(districtScore(after[i]))})
	}
	r.explain()
	return r, nil
}
func round(v float64) float64 { return math.Round(v*100) / 100 }
func (r *Result) explain() {
	sorted := append([]DistrictResult{}, r.Districts...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Score < sorted[j].Score })
	r.Risks = []string{fmt.Sprintf("Самый уязвимый район — %s (%.2f).", sorted[0].Name, sorted[0].Score)}
	if r.CriticalCount > 0 {
		r.Risks = append(r.Risks, fmt.Sprintf("Критических показателей ниже 40: %d; они снижают итоговый балл.", r.CriticalCount))
	}
	if r.Delta >= 0 {
		r.Strengths = []string{fmt.Sprintf("Сценарий улучшает городской балл на %.2f пункта.", r.Delta)}
	} else {
		r.Risks = append(r.Risks, "Сценарий ухудшает базовый результат.")
	}
	r.Recommendations = []string{"Направляйте следующую меру в слабейший район: формула учитывает минимум по районам на 30%."}
	if r.Remaining > 0 {
		r.Recommendations = append(r.Recommendations, fmt.Sprintf("Осталось %d ед. бюджета: это не даёт бонуса, но позволяет заменить менее эффективную меру.", r.Remaining))
	}
	r.Strengths = append(r.Strengths, "Расчёт детерминирован: AI может объяснять результат, но не меняет числа.")
}
func Directions() []string { return []string{"transport", "green", "social", "safety", "services"} }
