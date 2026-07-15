package user

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID              `json:"id"`
	UUID         uuid.UUID              `json:"uuid"`
	Email        string                 `json:"email"`
	Name         string                 `json:"name"`
	Avatar       string                 `json:"avatar,omitempty"`
	Role         string                 `json:"role"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	LastLogin    *time.Time             `json:"last_login,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

type CreateUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}
