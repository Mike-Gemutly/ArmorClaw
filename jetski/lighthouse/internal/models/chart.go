package models

// Chart represents a chart in the Lighthouse system
type Chart struct {
	ID        int    `json:"id"`
	Domain    string `json:"domain"`
	Version   string `json:"version"`
	Author    string `json:"author"`
	ChartData string `json:"chartData"`
	Signature string `json:"signature,omitempty"`
	Blessed   bool   `json:"blessed"`
	Downloads int    `json:"downloads"`
	CreatedAt string `json:"createdAt"`
}
