package model

// ScopedItem is a single affected thing (service, region, customer, API) along
// with the engine's confidence that it was actually impacted.
type ScopedItem struct {
	Name       string  `json:"name"`
	Confidence float64 `json:"confidence"`
}

// BlastRadius estimates the scope of impact of the incident.
type BlastRadius struct {
	Services  []ScopedItem `json:"services"`
	Regions   []ScopedItem `json:"regions"`
	Customers []ScopedItem `json:"customers"`
	APIs      []ScopedItem `json:"apis"`
}
