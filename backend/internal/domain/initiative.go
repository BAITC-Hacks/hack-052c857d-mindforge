package domain

// InitiativeCategory identifies the area an initiative belongs to.
type InitiativeCategory string

const (
	CategoryTransport InitiativeCategory = "transport"
	CategoryGreen     InitiativeCategory = "green"
	CategorySocial    InitiativeCategory = "social"
	CategorySafety    InitiativeCategory = "safety"
	CategoryServices  InitiativeCategory = "services"
)

// InitiativeEffects describes changes to the five district indicators.
type InitiativeEffects struct {
	Transport float64 `json:"transport"`
	Green     float64 `json:"green"`
	Social    float64 `json:"social"`
	Safety    float64 `json:"safety"`
	Services  float64 `json:"services"`
}

// Initiative is an available city improvement with a cost and expected effects.
type Initiative struct {
	ID       string             `json:"id"`
	Name     string             `json:"name"`
	Category InitiativeCategory `json:"category"`
	Cost     int64              `json:"cost"`
	Effects  InitiativeEffects  `json:"effects"`
}
