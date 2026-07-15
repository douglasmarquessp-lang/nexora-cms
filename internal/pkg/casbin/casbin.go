package casbin

import (
	"context"
	"embed"
	"fmt"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"nexora/internal/pkg/logger"
)

//go:embed model.conf
var modelFS embed.FS

type pgxAdapter struct {
	pool *pgxpool.Pool
}

func newPgxAdapter(pool *pgxpool.Pool) *pgxAdapter {
	return &pgxAdapter{pool: pool}
}

func (a *pgxAdapter) LoadPolicy(m model.Model) error {
	rows, err := a.pool.Query(context.Background(),
		`SELECT ptype, COALESCE(v0,''), COALESCE(v1,''), COALESCE(v2,''), COALESCE(v3,''), COALESCE(v4,''), COALESCE(v5,'')
		 FROM casbin_rules`,
	)
	if err != nil {
		return fmt.Errorf("failed to load casbin rules: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var ptype, v0, v1, v2, v3, v4, v5 string
		if err := rows.Scan(&ptype, &v0, &v1, &v2, &v3, &v4, &v5); err != nil {
			return fmt.Errorf("failed to scan casbin rule: %w", err)
		}

		rule := []string{ptype}
		if v0 != "" {
			rule = append(rule, v0)
		}
		if v1 != "" {
			rule = append(rule, v1)
		}
		if v2 != "" {
			rule = append(rule, v2)
		}
		if v3 != "" {
			rule = append(rule, v3)
		}
		if v4 != "" {
			rule = append(rule, v4)
		}
		if v5 != "" {
			rule = append(rule, v5)
		}

		persist.LoadPolicyArray(rule, m)
	}

	return nil
}

func (a *pgxAdapter) SavePolicy(m model.Model) error {
	if err := a.dropAllRules(); err != nil {
		return err
	}

	var rules []ruleEntry

	for ptype, assertions := range m["p"] {
		for _, rule := range assertions.Policy {
			entry := ruleEntry{ptype: ptype}
			for i, v := range rule {
				switch i {
				case 0:
					entry.v0 = v
				case 1:
					entry.v1 = v
				case 2:
					entry.v2 = v
				case 3:
					entry.v3 = v
				case 4:
					entry.v4 = v
				case 5:
					entry.v5 = v
				}
			}
			rules = append(rules, entry)
		}
	}

	for ptype, assertions := range m["g"] {
		for _, rule := range assertions.Policy {
			entry := ruleEntry{ptype: ptype}
			for i, v := range rule {
				switch i {
				case 0:
					entry.v0 = v
				case 1:
					entry.v1 = v
				case 2:
					entry.v2 = v
				case 3:
					entry.v3 = v
				case 4:
					entry.v4 = v
				case 5:
					entry.v5 = v
				}
			}
			rules = append(rules, entry)
		}
	}

	return a.insertRules(rules)
}

type ruleEntry struct {
	ptype    string
	v0, v1, v2, v3, v4, v5 string
}

func (a *pgxAdapter) dropAllRules() error {
	_, err := a.pool.Exec(context.Background(), `DELETE FROM casbin_rules`)
	return err
}

func (a *pgxAdapter) insertRules(rules []ruleEntry) error {
	if len(rules) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	for _, r := range rules {
		batch.Queue(
			`INSERT INTO casbin_rules (ptype, v0, v1, v2, v3, v4, v5) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
			r.ptype, r.v0, r.v1, r.v2, r.v3, r.v4, r.v5,
		)
	}

	br := a.pool.SendBatch(context.Background(), batch)
	defer br.Close()

	for range rules {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("failed to insert casbin rule: %w", err)
		}
	}

	return nil
}

func (a *pgxAdapter) AddPolicy(_ string, ptype string, rule []string) error {
	return a.insertRule(ptype, rule)
}

func (a *pgxAdapter) RemovePolicy(_ string, ptype string, rule []string) error {
	return a.deleteRule(ptype, rule)
}

func (a *pgxAdapter) RemoveFilteredPolicy(_ string, ptype string, fieldIndex int, fieldValues ...string) error {
	where := fmt.Sprintf("ptype = $1")
	args := []interface{}{ptype}
	argIdx := 2

	for i, v := range fieldValues {
		if v == "" {
			continue
		}
		col := fmt.Sprintf("v%d", fieldIndex+i)
		where += fmt.Sprintf(" AND %s = $%d", col, argIdx)
		args = append(args, v)
		argIdx++
	}

	query := fmt.Sprintf("DELETE FROM casbin_rules WHERE %s", where)
	_, err := a.pool.Exec(context.Background(), query, args...)
	return err
}

func (a *pgxAdapter) insertRule(ptype string, rule []string) error {
	v0, v1, v2, v3, v4, v5 := "", "", "", "", "", ""
	for i, v := range rule {
		switch i {
		case 0:
			v0 = v
		case 1:
			v1 = v
		case 2:
			v2 = v
		case 3:
			v3 = v
		case 4:
			v4 = v
		case 5:
			v5 = v
		}
	}

	_, err := a.pool.Exec(context.Background(),
		`INSERT INTO casbin_rules (ptype, v0, v1, v2, v3, v4, v5) VALUES ($1,$2,$3,$4,$5,$6,$7)
		 ON CONFLICT DO NOTHING`,
		ptype, v0, v1, v2, v3, v4, v5,
	)
	return err
}

func (a *pgxAdapter) deleteRule(ptype string, rule []string) error {
	v0, v1, v2, v3, v4, v5 := "", "", "", "", "", ""
	for i, v := range rule {
		switch i {
		case 0:
			v0 = v
		case 1:
			v1 = v
		case 2:
			v2 = v
		case 3:
			v3 = v
		case 4:
			v4 = v
		case 5:
			v5 = v
		}
	}

	_, err := a.pool.Exec(context.Background(),
		`DELETE FROM casbin_rules WHERE ptype = $1 AND v0 = $2 AND v1 = $3 AND v2 = $4 AND v3 = $5 AND v4 = $6 AND v5 = $7`,
		ptype, v0, v1, v2, v3, v4, v5,
	)
	return err
}

type Enforcer struct {
	enforcer *casbin.Enforcer
	log      *logger.Logger
}

func New(db *pgxpool.Pool, log *logger.Logger) (*Enforcer, error) {
	modelData, err := modelFS.ReadFile("model.conf")
	if err != nil {
		return nil, fmt.Errorf("failed to read model.conf: %w", err)
	}

	m, err := model.NewModelFromString(string(modelData))
	if err != nil {
		return nil, fmt.Errorf("failed to create casbin model: %w", err)
	}

	adapter := newPgxAdapter(db)

	e, err := casbin.NewEnforcer(m, adapter)
	if err != nil {
		return nil, fmt.Errorf("failed to create casbin enforcer: %w", err)
	}

	if err := e.LoadPolicy(); err != nil {
		return nil, fmt.Errorf("failed to load casbin policy: %w", err)
	}

	return &Enforcer{
		enforcer: e,
		log:      log,
	}, nil
}

func strToInterface(s []string) []interface{} {
	result := make([]interface{}, len(s))
	for i, v := range s {
		result[i] = v
	}
	return result
}

func (e *Enforcer) Enforce(sub, domain, obj, act string) (bool, error) {
	ok, err := e.enforcer.Enforce(sub, domain, obj, act)
	if err != nil {
		return false, fmt.Errorf("casbin enforce error: %w", err)
	}
	return ok, nil
}

func (e *Enforcer) AddPolicy(ptype string, params []string) error {
	converted := strToInterface(params)
	added, err := e.enforcer.AddPolicy(converted...)
	if err != nil {
		return fmt.Errorf("failed to add policy: %w", err)
	}
	if !added {
		return fmt.Errorf("policy already exists")
	}
	return nil
}

func (e *Enforcer) RemovePolicy(ptype string, params []string) error {
	converted := strToInterface(params)
	removed, err := e.enforcer.RemovePolicy(converted...)
	if err != nil {
		return fmt.Errorf("failed to remove policy: %w", err)
	}
	if !removed {
		return fmt.Errorf("policy not found")
	}
	return nil
}

func (e *Enforcer) GetRolesForUser(user, domain string) ([]string, error) {
	// In casbin v2, GetRolesForUserInDomain returns just []string
	roles := e.enforcer.GetRolesForUserInDomain(user, domain)
	return roles, nil
}

func (e *Enforcer) GetUsersForRole(role, domain string) ([]string, error) {
	users := e.enforcer.GetUsersForRoleInDomain(role, domain)
	return users, nil
}

func (e *Enforcer) AddRoleForUser(user, role, domain string) error {
	added, err := e.enforcer.AddRoleForUserInDomain(user, role, domain)
	if err != nil {
		return fmt.Errorf("failed to add role for user: %w", err)
	}
	if !added {
		return fmt.Errorf("role mapping already exists")
	}
	return nil
}

func (e *Enforcer) RemoveRoleForUser(user, role, domain string) error {
	removed, err := e.enforcer.DeleteRoleForUserInDomain(user, role, domain)
	if err != nil {
		return fmt.Errorf("failed to remove role for user: %w", err)
	}
	if !removed {
		return fmt.Errorf("role mapping not found")
	}
	return nil
}

// Ensure the adapter implements the persist.Adapter interface
var _ persist.Adapter = (*pgxAdapter)(nil)
