package domain

import "time"

type User struct {
	ID           string    `json:"id"`
	Phone        string    `json:"phone"`
	Email        string    `json:"email,omitempty"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	Nick         string    `json:"nick"`
	PasswordHash string    `json:"-"`
	Roles        []string  `json:"roles"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}
