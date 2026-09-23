// Package simulation contains the deterministic, auditable city model.
// It deliberately does not call an LLM: an LLM may explain this output, but
// all scores must stay reproducible.
package simulation

import (
	"fmt"
	"math"
	"sort"
)

const Budget = 100

type Metrics struct{ T1, T2, E1, E2, S1, S2, B1, B2, C1, C2 float64 }
type District struct {
	ID, Name, Profile string
	Population        float64
	Metrics           Metrics
}
type Initiative struct {
	ID, Direction, Name, Scope string
	Cost, Lag                  int
	Effects                    Metrics
}
type Choice struct {
	InitiativeID string `json:"initiative_id"`
	DistrictID   string `json:"district_id,omitempty"`
}
type DistrictResult struct {
	ID, Name      string
	Before, After Metrics
	Score         float64
}
type Result struct {
	Budget, Spent, Remaining          int
	Score, BaselineScore, Delta       float64
	CriticalCount                     int
	Districts                         []DistrictResult
	Strengths, Risks, Recommendations []string
}

var Districts = []District{
	{"esil", "Есиль", "Пробки на мостах и переполненные школы.", .27, Metrics{45, 62, 68, 72, 48, 55, 78, 60, 75, 70}},
	{"almaty", "Алматы", "Старый ЖКХ и пробки.", .24, Metrics{40, 75, 50, 55, 60, 65, 62, 52, 50, 60}},
	{"saryarka", "Сарыарка", "Смог от частного сектора, слабое озеленение.", .20, Metrics{50, 70, 42, 40, 62, 68, 58, 55, 45, 55}},
	{"baikonyur", "Байконур", "Район без ярких перекосов.", .13, Metrics{52, 68, 55, 50, 58, 60, 52, 58, 55, 58}},
	{"nura", "Нура", "Главный аутсайдер по соцсфере и транспорту.", .16, Metrics{55, 40, 45, 65, 38, 35, 55, 50, 60, 50}},
}
var Initiatives = []Initiative{
	{"M1", "transport", "Выделенные полосы для автобусов", "district", 18, 2, Metrics{T1: 6, T2: 9}},
	{"M2", "transport", "Умные светофоры", "city", 22, 2, Metrics{T1: 4, B2: 3}},
	{"M3", "transport", "Линия ЛРТ / расширение", "district", 30, 4, Metrics{T1: 16, T2: 20, E2: 4}},
	{"M4", "green", "Парк / сквер", "district", 15, 2, Metrics{E1: 12, E2: 3, B1: 2}},
	{"M5", "green", "Чистое топливо для частного сектора", "district", 25, 3, Metrics{E2: 14, C1: 4}},
	{"M6", "green", "Озеленение и ветрозащитные полосы", "city", 20, 4, Metrics{E1: 5, E2: 3}},
	{"M7", "social", "Школа + детсад", "district", 24, 3, Metrics{S1: 16}},
	{"M8", "social", "Центр семейного здоровья / поликлиника", "district", 20, 3, Metrics{S2: 14}},
	{"M9", "social", "Дворовые спорт-хабы", "district", 10, 1, Metrics{S1: 3, S2: 3, B1: 3}},
	{"M10", "safety", "Освещение и камеры Safe City", "district", 12, 1, Metrics{B1: 12, B2: 2}},
	{"M11", "safety", "Безопасные переходы и школьные зоны", "district", 10, 1, Metrics{T1: -2, B2: 12}},
	{"M12", "services", "Цифровая платформа обращений", "city", 14, 1, Metrics{C2: 5}},
	{"M13", "services", "Модернизация тепло- и водосетей", "district", 28, 4, Metrics{C1: 18, E2: 2}},
	{"M14", "services", "Аварийные бригады ЖКХ", "city", 16, 1, Metrics{C1: 5, C2: 2}},
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
