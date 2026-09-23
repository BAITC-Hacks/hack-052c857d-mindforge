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
	"strings"
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
type Synergy struct {
	Initiatives []string `json:"initiatives"`
	District    string   `json:"district"`
	Indicator   string   `json:"indicator"`
	Bonus       float64  `json:"bonus"`
}
type DistrictResult struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	PopulationShare float64 `json:"population_share"`
	Before          Metrics `json:"before"`
	After           Metrics `json:"after"`
	Score           float64 `json:"score"`
	Delta           float64 `json:"delta"`
}
type Result struct {
	Budget           int              `json:"budget"`
	Spent            int              `json:"spent"`
	Remaining        int              `json:"remaining"`
	Score            float64          `json:"score"`
	BaselineScore    float64          `json:"baseline_score"`
	Delta            float64          `json:"delta"`
	CriticalCount    int              `json:"critical_count"`
	BaselineCritical int              `json:"baseline_critical_count,omitempty"`
	Selected         []Choice         `json:"selected,omitempty"`
	Districts        []DistrictResult `json:"districts"`
	Strengths        []string         `json:"strengths"`
	Risks            []string         `json:"risks"`
	Recommendations  []string         `json:"recommendations"`
	Synergies        []Synergy        `json:"synergies,omitempty"`
	Valid            bool             `json:"valid"`
	ValidationErrors []string         `json:"validation_errors,omitempty"`
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

func normalizeDistrictID(id string) string {
	return strings.ToLower(strings.TrimSpace(id))
}
func normalizeInitiativeID(id string) string {
	return strings.ToUpper(strings.TrimSpace(id))
}

func catalog() map[string]Initiative {
	r := map[string]Initiative{}
	for _, x := range Initiatives {
		r[normalizeInitiativeID(x.ID)] = x
	}
	return r
}
func findDistrict(id string) (int, bool) {
	normalized := normalizeDistrictID(id)
	for i, d := range Districts {
		if normalizeDistrictID(d.ID) == normalized {
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
	avg, minScore := 0.0, math.Inf(1)
	crit := 0
	for i, m := range ms {
		s := districtScore(m)
		avg += Districts[i].Population * s
		if s < minScore {
			minScore = s
		}
		crit += critical(m)
	}
	return .7*avg + .3*minScore - float64(crit), crit
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
		id := normalizeInitiativeID(c.InitiativeID)
		x, ok := cat[id]
		if !ok {
			return nil, fmt.Errorf("неизвестное мероприятие %q", c.InitiativeID)
		}
		c.InitiativeID = x.ID
		c.DistrictID = normalizeDistrictID(c.DistrictID)
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
			if c.DistrictID == "" {
				return nil, fmt.Errorf("для %s укажите район", x.ID)
			}
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
		if seen[pair[0]] && seen[pair[1]] && normalizeDistrictID(chosen[pair[0]].DistrictID) == normalizeDistrictID(chosen[pair[1]].DistrictID) {
			return nil, fmt.Errorf("%s и %s нельзя размещать в одном районе", pair[0], pair[1])
		}
	}
	items := make([]Initiative, 0, 5)
	for _, c := range choices {
		items = append(items, cat[normalizeInitiativeID(c.InitiativeID)])
	}
	return items, nil
}

func Simulate(choices []Choice) (Result, error) {
	items, err := Validate(choices)
	if err != nil {
		return Result{Valid: false, ValidationErrors: []string{err.Error()}}, err
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
	lookup := map[string]Choice{}
	for _, c := range choices {
		lookup[normalizeInitiativeID(c.InitiativeID)] = c
	}
	synergies := make([]Synergy, 0)
	for _, s := range []struct {
		a, b      string
		indicator string
		bonus     float64
	}{{"M1", "M2", "T1", 2}, {"M10", "M12", "B1", 2}, {"M5", "M6", "E2", 2}} {
		if _, ok := lookup[s.a]; ok {
			if _, ok2 := lookup[s.b]; ok2 {
				districtID := ""
				switch s.a {
				case "M1":
					districtID = lookup["M1"].DistrictID
				case "M10":
					districtID = lookup["M10"].DistrictID
				case "M5":
					districtID = lookup["M5"].DistrictID
				}
				if districtID == "" {
					continue
				}
				d, found := findDistrict(districtID)
				if !found {
					continue
				}
				synergies = append(synergies, Synergy{Initiatives: []string{s.a, s.b}, District: districtID, Indicator: s.indicator, Bonus: s.bonus})
				switch s.indicator {
				case "T1":
					after[d].T1 += s.bonus
				case "B1":
					after[d].B1 += s.bonus
				case "E2":
					after[d].E2 += s.bonus
				}
			}
		}
	}
	for i := range after {
		clip(&after[i])
	}
	score, crit := cityScore(after)
	baseScore, baseCrit := cityScore(before)
	r := Result{
		Budget:           Budget,
		Spent:            spent,
		Remaining:        Budget - spent,
		Score:            round(score),
		BaselineScore:    round(baseScore),
		Delta:            round(score - baseScore),
		CriticalCount:    crit,
		BaselineCritical: baseCrit,
		Selected:         append([]Choice(nil), choices...),
		Valid:            true,
		Synergies:        synergies,
	}
	for i, d := range Districts {
		r.Districts = append(r.Districts, DistrictResult{
			ID:              d.ID,
			Name:            d.Name,
			PopulationShare: d.Population,
			Before:          before[i],
			After:           after[i],
			Score:           round(districtScore(after[i])),
			Delta:           round(districtScore(after[i]) - districtScore(before[i])),
		})
	}
	r.explain()
	return r, nil
}
func round(v float64) float64 { return math.Round(v*100) / 100 }
func (r *Result) explain() {
	sorted := append([]DistrictResult{}, r.Districts...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Score < sorted[j].Score })
	if len(sorted) > 0 {
		r.Risks = []string{fmt.Sprintf("Самый уязвимый район — %s (%.2f).", sorted[0].Name, sorted[0].Score)}
	} else {
		r.Risks = []string{"Нет данных по районам."}
	}
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
