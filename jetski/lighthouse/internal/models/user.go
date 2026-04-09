package models

// User represents a user in the Lighthouse system
type User struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	APIKey    string `json:"apiKey"`
	Role      string `json:"role"`
	CreatedAt string `json:"createdAt"`
}
