package domain

// DistrictMetrics holds the five indicators used by the city simulator.
type DistrictMetrics struct {
	Transport float64 `json:"transport"`
	Green     float64 `json:"green"`
	Social    float64 `json:"social"`
	Safety    float64 `json:"safety"`
	Services  float64 `json:"services"`
}

// District is a city district and its current indicator values.
type District struct {
	ID      string          `json:"id"`
	Name    string          `json:"name"`
	Metrics DistrictMetrics `json:"metrics"`
}
