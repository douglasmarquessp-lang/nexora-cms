package audit

import (
	"context"
	"time"

	"github.com/google/uuid"

	"nexora/internal/pkg/database"
	"nexora/internal/pkg/logger"
)

type Action string

const (
	ActionUserLogin       Action = "user.login"
	ActionUserLogout      Action = "user.logout"
	ActionUserRegistered  Action = "user.registered"
	ActionOAuthLinked     Action = "oauth.linked"
	ActionOAuthLogin      Action = "oauth.login"
	ActionMFARegistered   Action = "mfa.registered"
	ActionMFAVerified     Action = "mfa.verified"
	ActionMFADisabled     Action = "mfa.disabled"
	ActionTokenRefreshed  Action = "token.refreshed"
	ActionPasswordChanged Action = "password.changed"
	ActionUserUpdated     Action = "user.updated"
	ActionUserDeleted     Action = "user.deleted"
)

type Entry struct {
	ID         uuid.UUID              `json:"id"`
	UserID     *uuid.UUID             `json:"user_id,omitempty"`
	SiteID     *uuid.UUID             `json:"site_id,omitempty"`
	Action     Action                 `json:"action"`
	EntityType string                 `json:"entity_type"`
	EntityID   *uuid.UUID             `json:"entity_id,omitempty"`
	Payload    map[string]interface{} `json:"payload,omitempty"`
	IPAddress  string                 `json:"ip_address,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
}

type Logger struct {
	pool database.Pool
	log  *logger.Logger
}

func New(pool database.Pool, log *logger.Logger) *Logger {
	return &Logger{pool: pool, log: log}
}

func (l *Logger) Log(ctx context.Context, entry Entry) {
	if l.pool == nil {
		l.log.Warn("audit log skipped: no database connection",
			"action", entry.Action,
		)
		return
	}

	userID := entry.UserID
	if userID != nil && *userID == uuid.Nil {
		userID = nil
	}

	siteID := entry.SiteID
	if siteID != nil && *siteID == uuid.Nil {
		siteID = nil
	}

	payload := entry.Payload
	if payload == nil {
		payload = map[string]interface{}{}
	}

	_, err := l.pool.Exec(ctx,
		`INSERT INTO audit_log (user_id, site_id, action, entity_type, entity_id, payload, ip_address)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		userID, siteID, string(entry.Action),
		entry.EntityType, entry.EntityID, payload, entry.IPAddress,
	)
	if err != nil {
		l.log.Error("failed to write audit log",
			"action", entry.Action,
			"error", err,
		)
	}
}

func (l *Logger) LogUserAction(ctx context.Context, userID uuid.UUID, action Action, data map[string]interface{}) {
	l.Log(ctx, Entry{
		UserID:     &userID,
		Action:     action,
		EntityType: "user",
		EntityID:   &userID,
		Payload:    data,
	})
}
