package casbin

import (
	"fmt"
	"testing"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"

	"nexora/internal/pkg/config"
	"nexora/internal/pkg/logger"
)

func TestModelConf_Exists(t *testing.T) {
	data, err := modelFS.ReadFile("model.conf")
	if err != nil {
		t.Fatalf("failed to read model.conf: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("model.conf is empty")
	}
}

func TestModelConf_Valid(t *testing.T) {
	data, err := modelFS.ReadFile("model.conf")
	if err != nil {
		t.Fatalf("failed to read model.conf: %v", err)
	}

	m, err := model.NewModelFromString(string(data))
	if err != nil {
		t.Fatalf("failed to create model from model.conf: %v", err)
	}
	if m == nil {
		t.Fatal("expected non-nil model")
	}
}

func TestModelConf_Sections(t *testing.T) {
	data, err := modelFS.ReadFile("model.conf")
	if err != nil {
		t.Fatalf("failed to read model.conf: %v", err)
	}

	m, err := model.NewModelFromString(string(data))
	if err != nil {
		t.Fatalf("failed to create model: %v", err)
	}

	requiredSections := []string{"r", "p", "g", "e", "m"}
	for _, sec := range requiredSections {
		if _, ok := m[sec]; !ok {
			t.Errorf("missing section '%s' in model.conf", sec)
		}
	}
}

func TestNew_NilPool(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)

	defer func() {
		if r := recover(); r != nil {
			// Expected: New with nil pool panics when adapter tries to query
		}
	}()

	_, err := New(nil, log)
	if err == nil {
		t.Fatal("expected error when pool is nil")
	}
}

func TestEnforcer_Enforce_Signature(t *testing.T) {
	var e *Enforcer
	var ok bool
	var err error

	_ = func() {
		ok, err = e.Enforce("sub", "dom", "obj", "act")
	}
	_ = ok
	_ = err
}

func TestEnforcer_AddPolicy_Signature(t *testing.T) {
	var e *Enforcer
	var err error

	_ = func() {
		err = e.AddPolicy("p", []string{"sub", "dom", "obj", "act"})
	}
	_ = err
}

func TestEnforcer_RemovePolicy_Signature(t *testing.T) {
	var e *Enforcer
	var err error

	_ = func() {
		err = e.RemovePolicy("p", []string{"sub", "dom", "obj", "act"})
	}
	_ = err
}

func TestEnforcer_GetRolesForUser_Signature(t *testing.T) {
	var e *Enforcer
	var roles []string
	var err error

	_ = func() {
		roles, err = e.GetRolesForUser("user", "domain")
	}
	_ = roles
	_ = err
}

func TestEnforcer_GetUsersForRole_Signature(t *testing.T) {
	var e *Enforcer
	var users []string
	var err error

	_ = func() {
		users, err = e.GetUsersForRole("admin", "domain")
	}
	_ = users
	_ = err
}

func TestEnforcer_AddRoleForUser_Signature(t *testing.T) {
	var e *Enforcer
	var err error

	_ = func() {
		err = e.AddRoleForUser("user", "admin", "domain")
	}
	_ = err
}

func TestEnforcer_RemoveRoleForUser_Signature(t *testing.T) {
	var e *Enforcer
	var err error

	_ = func() {
		err = e.RemoveRoleForUser("user", "admin", "domain")
	}
	_ = err
}

func TestPgxAdapter_ImplementsAdapter(t *testing.T) {
	var _ persist.Adapter = (*pgxAdapter)(nil)
}

const testModel = `
[request_definition]
r = sub, dom, obj, act

[policy_definition]
p = sub, dom, obj, act

[role_definition]
g = _, _, _
g2 = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub, r.dom) && r.dom == p.dom && keyMatch(r.obj, p.obj) && (r.act == p.act || p.act == "*")
`

func newTestEnforcer(t *testing.T) *casbin.Enforcer {
	t.Helper()
	m, err := model.NewModelFromString(testModel)
	if err != nil {
		t.Fatalf("failed to create model: %v", err)
	}
	e, err := casbin.NewEnforcer(m)
	if err != nil {
		t.Fatalf("failed to create enforcer: %v", err)
	}
	return e
}

func TestEnforcer_Enforce_SuperAdmin(t *testing.T) {
	e := newTestEnforcer(t)

	_, _ = e.AddRoleForUserInDomain("superadmin_user", "superadmin", "domain1")
	_, _ = e.AddPolicy("superadmin", "domain1", "*", "*")

	tests := []struct {
		sub, dom, obj, act string
		want               bool
	}{
		{"superadmin_user", "domain1", "post", "read", true},
		{"superadmin_user", "domain1", "post", "write", true},
		{"superadmin_user", "domain1", "post", "delete", true},
		{"superadmin_user", "domain1", "settings", "manage", true},
		{"superadmin_user", "domain1", "anything", "anything", true},
		{"superadmin_user", "domain2", "post", "read", false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s/%s/%s/%s", tt.sub, tt.dom, tt.obj, tt.act), func(t *testing.T) {
			got, err := e.Enforce(tt.sub, tt.dom, tt.obj, tt.act)
			if err != nil {
				t.Fatalf("enforce error: %v", err)
			}
			if got != tt.want {
				t.Errorf("Enforce(%q,%q,%q,%q) = %v, want %v", tt.sub, tt.dom, tt.obj, tt.act, got, tt.want)
			}
		})
	}
}

func TestEnforcer_Enforce_SiteAdmin(t *testing.T) {
	e := newTestEnforcer(t)

	_, _ = e.AddRoleForUserInDomain("siteadmin_user", "siteadmin", "site1")
	_, _ = e.AddPolicy("siteadmin", "site1", "*", "*")

	tests := []struct {
		sub, dom, obj, act string
		want               bool
	}{
		{"siteadmin_user", "site1", "post", "read", true},
		{"siteadmin_user", "site1", "post", "write", true},
		{"siteadmin_user", "site1", "post", "delete", true},
		{"siteadmin_user", "site1", "settings", "manage", true},
		{"siteadmin_user", "site2", "post", "read", false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s/%s/%s/%s", tt.sub, tt.dom, tt.obj, tt.act), func(t *testing.T) {
			got, err := e.Enforce(tt.sub, tt.dom, tt.obj, tt.act)
			if err != nil {
				t.Fatalf("enforce error: %v", err)
			}
			if got != tt.want {
				t.Errorf("Enforce(%q,%q,%q,%q) = %v, want %v", tt.sub, tt.dom, tt.obj, tt.act, got, tt.want)
			}
		})
	}
}

func TestEnforcer_Enforce_Editor(t *testing.T) {
	e := newTestEnforcer(t)

	_, _ = e.AddRoleForUserInDomain("editor_user", "editor", "blog")
	_, _ = e.AddPolicy("editor", "blog", "post", "write")
	_, _ = e.AddPolicy("editor", "blog", "post", "read")
	_, _ = e.AddPolicy("editor", "blog", "post", "delete")

	tests := []struct {
		sub, dom, obj, act string
		want               bool
	}{
		{"editor_user", "blog", "post", "read", true},
		{"editor_user", "blog", "post", "write", true},
		{"editor_user", "blog", "post", "delete", true},
		{"editor_user", "blog", "settings", "write", false},
		{"editor_user", "other", "post", "read", false},
		{"editor_user", "blog", "post", "publish", false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s/%s/%s/%s", tt.sub, tt.dom, tt.obj, tt.act), func(t *testing.T) {
			got, err := e.Enforce(tt.sub, tt.dom, tt.obj, tt.act)
			if err != nil {
				t.Fatalf("enforce error: %v", err)
			}
			if got != tt.want {
				t.Errorf("Enforce(%q,%q,%q,%q) = %v, want %v", tt.sub, tt.dom, tt.obj, tt.act, got, tt.want)
			}
		})
	}
}

func TestEnforcer_Enforce_Author(t *testing.T) {
	e := newTestEnforcer(t)

	_, _ = e.AddRoleForUserInDomain("author_user", "author", "myblog")
	_, _ = e.AddPolicy("author", "myblog", "post", "create")
	_, _ = e.AddPolicy("author", "myblog", "post", "read")

	tests := []struct {
		sub, dom, obj, act string
		want               bool
	}{
		{"author_user", "myblog", "post", "create", true},
		{"author_user", "myblog", "post", "read", true},
		{"author_user", "myblog", "post", "write", false},
		{"author_user", "myblog", "post", "delete", false},
		{"author_user", "myblog", "settings", "read", false},
		{"author_user", "other", "post", "create", false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s/%s/%s/%s", tt.sub, tt.dom, tt.obj, tt.act), func(t *testing.T) {
			got, err := e.Enforce(tt.sub, tt.dom, tt.obj, tt.act)
			if err != nil {
				t.Fatalf("enforce error: %v", err)
			}
			if got != tt.want {
				t.Errorf("Enforce(%q,%q,%q,%q) = %v, want %v", tt.sub, tt.dom, tt.obj, tt.act, got, tt.want)
			}
		})
	}
}

func TestEnforcer_Enforce_Subscriber(t *testing.T) {
	e := newTestEnforcer(t)

	_, _ = e.AddRoleForUserInDomain("sub_user", "subscriber", "news")
	_, _ = e.AddPolicy("subscriber", "news", "post", "read")

	tests := []struct {
		sub, dom, obj, act string
		want               bool
	}{
		{"sub_user", "news", "post", "read", true},
		{"sub_user", "news", "post", "write", false},
		{"sub_user", "news", "post", "create", false},
		{"sub_user", "news", "post", "delete", false},
		{"sub_user", "news", "comment", "read", false},
		{"sub_user", "other", "post", "read", false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s/%s/%s/%s", tt.sub, tt.dom, tt.obj, tt.act), func(t *testing.T) {
			got, err := e.Enforce(tt.sub, tt.dom, tt.obj, tt.act)
			if err != nil {
				t.Fatalf("enforce error: %v", err)
			}
			if got != tt.want {
				t.Errorf("Enforce(%q,%q,%q,%q) = %v, want %v", tt.sub, tt.dom, tt.obj, tt.act, got, tt.want)
			}
		})
	}
}

func TestEnforcer_Enforce_UnauthenticatedUser(t *testing.T) {
	e := newTestEnforcer(t)

	_, _ = e.AddPolicy("editor", "site1", "post", "write")

	got, err := e.Enforce("unknown_user", "site1", "post", "write")
	if err != nil {
		t.Fatalf("enforce error: %v", err)
	}
	if got {
		t.Error("expected unauthenticated user to be denied")
	}
}

func TestEnforcer_Enforce_MultipleRoles(t *testing.T) {
	e := newTestEnforcer(t)

	_, _ = e.AddRoleForUserInDomain("multi_user", "subscriber", "site")
	_, _ = e.AddRoleForUserInDomain("multi_user", "editor", "site")
	_, _ = e.AddPolicy("subscriber", "site", "post", "read")
	_, _ = e.AddPolicy("editor", "site", "post", "write")

	okRead, _ := e.Enforce("multi_user", "site", "post", "read")
	if !okRead {
		t.Error("expected multi-role user to read")
	}
	okWrite, _ := e.Enforce("multi_user", "site", "post", "write")
	if !okWrite {
		t.Error("expected multi-role user to write")
	}
}

func TestEnforcer_Enforce_PolicyWildcardAction(t *testing.T) {
	e := newTestEnforcer(t)

	_, _ = e.AddRoleForUserInDomain("wild_user", "admin", "s1")
	_, _ = e.AddPolicy("admin", "s1", "post", "*")

	tests := []struct {
		act  string
		want bool
	}{
		{"read", true},
		{"write", true},
		{"delete", true},
		{"publish", true},
	}
	for _, tt := range tests {
		t.Run(tt.act, func(t *testing.T) {
			got, err := e.Enforce("wild_user", "s1", "post", tt.act)
			if err != nil {
				t.Fatalf("enforce error: %v", err)
			}
			if got != tt.want {
				t.Errorf("expected %v for act=%q, got %v", tt.want, tt.act, got)
			}
		})
	}
}

func TestEnforcer_Enforce_KeyMatchObject(t *testing.T) {
	e := newTestEnforcer(t)

	_, _ = e.AddRoleForUserInDomain("key_user", "admin", "d1")
	_, _ = e.AddPolicy("admin", "d1", "/api/posts/*", "read")

	tests := []struct {
		obj  string
		want bool
	}{
		{"/api/posts/list", true},
		{"/api/posts/123", true},
		{"/api/posts/", true},
		{"/api/users/list", false},
		{"post", false},
	}
	for _, tt := range tests {
		t.Run(tt.obj, func(t *testing.T) {
			got, err := e.Enforce("key_user", "d1", tt.obj, "read")
			if err != nil {
				t.Fatalf("enforce error: %v", err)
			}
			if got != tt.want {
				t.Errorf("Enforce obj=%q: got %v, want %v", tt.obj, got, tt.want)
			}
		})
	}
}

func TestStrToInterface(t *testing.T) {
	input := []string{"a", "b", "c"}
	result := strToInterface(input)
	if len(result) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(result))
	}
	for i, v := range result {
		s, ok := v.(string)
		if !ok || s != input[i] {
			t.Errorf("result[%d] = %v, expected %s", i, v, input[i])
		}
	}
}

func TestModelConf_SectionDetails(t *testing.T) {
	data, err := modelFS.ReadFile("model.conf")
	if err != nil {
		t.Fatalf("failed to read model.conf: %v", err)
	}

	m, err := model.NewModelFromString(string(data))
	if err != nil {
		t.Fatalf("failed to create model: %v", err)
	}

	if m["r"] == nil || m["r"]["r"] == nil {
		t.Fatal("missing request_definition")
	}
	if len(m["r"]["r"].Tokens) == 0 {
		t.Fatal("request_definition has no tokens")
	}

	if m["p"] == nil || m["p"]["p"] == nil {
		t.Fatal("missing policy_definition")
	}

	if m["g"] == nil || m["g"]["g"] == nil {
		t.Fatal("missing role_definition g")
	}
	if m["g"] == nil || m["g"]["g2"] == nil {
		t.Fatal("missing role_definition g2")
	}

	if m["e"] == nil || m["e"]["e"] == nil {
		t.Fatal("missing policy_effect")
	}

	if m["m"] == nil || m["m"]["m"] == nil {
		t.Fatal("missing matchers")
	}
}

func newWrapperEnforcer(t *testing.T) *Enforcer {
	t.Helper()
	m, err := model.NewModelFromString(testModel)
	if err != nil {
		t.Fatalf("failed to create model: %v", err)
	}
	enforcer, err := casbin.NewEnforcer(m)
	if err != nil {
		t.Fatalf("failed to create enforcer: %v", err)
	}
	cfg := &config.Config{}
	log := logger.New(cfg)
	return &Enforcer{
		enforcer: enforcer,
		log:      log,
	}
}

func TestWrapperEnforcer_Enforce(t *testing.T) {
	e := newWrapperEnforcer(t)
	e.enforcer.AddRoleForUserInDomain("testuser", "admin", "domain1")
	e.enforcer.AddPolicy("admin", "domain1", "*", "*")

	ok, err := e.Enforce("testuser", "domain1", "post", "read")
	if err != nil {
		t.Fatalf("Enforce failed: %v", err)
	}
	if !ok {
		t.Error("expected enforce to be allowed")
	}
}

func TestWrapperEnforcer_Enforce_Denied(t *testing.T) {
	e := newWrapperEnforcer(t)
	e.enforcer.AddRoleForUserInDomain("testuser", "subscriber", "domain1")
	e.enforcer.AddPolicy("subscriber", "domain1", "post", "read")

	ok, err := e.Enforce("testuser", "domain1", "post", "write")
	if err != nil {
		t.Fatalf("Enforce failed: %v", err)
	}
	if ok {
		t.Error("expected enforce to be denied")
	}
}

func TestWrapperEnforcer_AddPolicy(t *testing.T) {
	e := newWrapperEnforcer(t)
	e.enforcer.AddRoleForUserInDomain("u1", "editor", "d1")

	err := e.AddPolicy("p", []string{"editor", "d1", "post", "write"})
	if err != nil {
		t.Fatalf("AddPolicy failed: %v", err)
	}

	ok, _ := e.Enforce("u1", "d1", "post", "write")
	if !ok {
		t.Error("expected policy to be enforced")
	}
}

func TestWrapperEnforcer_AddPolicy_Duplicate(t *testing.T) {
	e := newWrapperEnforcer(t)

	err := e.AddPolicy("p", []string{"admin", "d1", "*", "*"})
	if err != nil {
		t.Fatalf("first AddPolicy failed: %v", err)
	}

	err = e.AddPolicy("p", []string{"admin", "d1", "*", "*"})
	if err == nil {
		t.Log("duplicate policy accepted (expected behavior)")
	}
}

func TestWrapperEnforcer_RemovePolicy(t *testing.T) {
	e := newWrapperEnforcer(t)
	e.enforcer.AddRoleForUserInDomain("u1", "role1", "d1")

	e.enforcer.AddPolicy("role1", "d1", "obj1", "act1")
	err := e.RemovePolicy("p", []string{"role1", "d1", "obj1", "act1"})
	if err != nil {
		t.Fatalf("RemovePolicy failed: %v", err)
	}

	ok, _ := e.Enforce("u1", "d1", "obj1", "act1")
	if ok {
		t.Error("expected policy to be removed")
	}
}

func TestWrapperEnforcer_RemovePolicy_NotFound(t *testing.T) {
	e := newWrapperEnforcer(t)

	err := e.RemovePolicy("p", []string{"nonexistent", "d1", "obj", "act"})
	if err == nil {
		t.Log("remove of nonexistent policy returned no error (expected)")
	}
}

func TestWrapperEnforcer_GetRolesForUser(t *testing.T) {
	e := newWrapperEnforcer(t)
	e.enforcer.AddRoleForUserInDomain("u1", "admin", "d1")

	roles, err := e.GetRolesForUser("u1", "d1")
	if err != nil {
		t.Fatalf("GetRolesForUser failed: %v", err)
	}
	if len(roles) != 1 || roles[0] != "admin" {
		t.Errorf("expected [admin], got %v", roles)
	}
}

func TestWrapperEnforcer_GetUsersForRole(t *testing.T) {
	e := newWrapperEnforcer(t)
	e.enforcer.AddRoleForUserInDomain("u1", "admin", "d1")
	e.enforcer.AddRoleForUserInDomain("u2", "admin", "d1")

	users, err := e.GetUsersForRole("admin", "d1")
	if err != nil {
		t.Fatalf("GetUsersForRole failed: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
}

func TestWrapperEnforcer_AddRoleForUser(t *testing.T) {
	e := newWrapperEnforcer(t)

	err := e.AddRoleForUser("u1", "editor", "d1")
	if err != nil {
		t.Fatalf("AddRoleForUser failed: %v", err)
	}

	e.enforcer.AddPolicy("editor", "d1", "post", "write")
	ok, _ := e.Enforce("u1", "d1", "post", "write")
	if !ok {
		t.Error("expected user to have editor access")
	}
}

func TestWrapperEnforcer_RemoveRoleForUser(t *testing.T) {
	e := newWrapperEnforcer(t)

	e.enforcer.AddRoleForUserInDomain("u1", "editor", "d1")
	err := e.RemoveRoleForUser("u1", "editor", "d1")
	if err != nil {
		t.Fatalf("RemoveRoleForUser failed: %v", err)
	}

	e.enforcer.AddPolicy("editor", "d1", "post", "write")
	ok, _ := e.Enforce("u1", "d1", "post", "write")
	if ok {
		t.Error("expected user to lose editor access")
	}
}

func TestWrapperEnforcer_AddRoleForUser_Duplicate(t *testing.T) {
	e := newWrapperEnforcer(t)

	err := e.AddRoleForUser("u1", "admin", "d1")
	if err != nil {
		t.Fatalf("first AddRoleForUser failed: %v", err)
	}

	err = e.AddRoleForUser("u1", "admin", "d1")
	if err == nil {
		t.Log("duplicate role accepted (expected behavior)")
	}
}

func TestWrapperEnforcer_RemoveRoleForUser_NotFound(t *testing.T) {
	e := newWrapperEnforcer(t)

	err := e.RemoveRoleForUser("u1", "nonexistent", "d1")
	if err == nil {
		t.Log("remove of nonexistent role returned no error (expected)")
	}
}

func TestAddAndRemovePolicy(t *testing.T) {
	e := newTestEnforcer(t)

	_, _ = e.AddRoleForUserInDomain("u1", "role1", "d1")
	_, _ = e.AddPolicy("role1", "d1", "obj1", "act1")

	ok, _ := e.Enforce("u1", "d1", "obj1", "act1")
	if !ok {
		t.Error("expected policy to be enforced")
	}

	_, _ = e.RemovePolicy("role1", "d1", "obj1", "act1")

	ok, _ = e.Enforce("u1", "d1", "obj1", "act1")
	if ok {
		t.Error("expected policy to no longer be enforced after removal")
	}
}

func TestAddAndRemoveRoleForUser(t *testing.T) {
	e := newTestEnforcer(t)

	_, _ = e.AddRoleForUserInDomain("u1", "editor", "d1")
	_, _ = e.AddPolicy("editor", "d1", "post", "write")

	ok, _ := e.Enforce("u1", "d1", "post", "write")
	if !ok {
		t.Error("expected user to have editor access")
	}

	_, _ = e.DeleteRoleForUserInDomain("u1", "editor", "d1")

	ok, _ = e.Enforce("u1", "d1", "post", "write")
	if ok {
		t.Error("expected user to lose editor access after role removal")
	}
}

func TestEnforcer_GetRolesForUserInDomain(t *testing.T) {
	e := newTestEnforcer(t)

	_, _ = e.AddRoleForUserInDomain("u1", "admin", "d1")
	_, _ = e.AddRoleForUserInDomain("u1", "editor", "d1")

	roles := e.GetRolesForUserInDomain("u1", "d1")
	if len(roles) != 2 {
		t.Fatalf("expected 2 roles, got %d: %v", len(roles), roles)
	}
}

func TestEnforcer_GetUsersForRoleInDomain(t *testing.T) {
	e := newTestEnforcer(t)

	_, _ = e.AddRoleForUserInDomain("u1", "admin", "d1")
	_, _ = e.AddRoleForUserInDomain("u2", "admin", "d1")

	users := e.GetUsersForRoleInDomain("admin", "d1")
	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d: %v", len(users), users)
	}
}

func TestEnforcer_GetRolesForUserInDomain_Empty(t *testing.T) {
	e := newTestEnforcer(t)

	roles := e.GetRolesForUserInDomain("unknown", "d1")
	if len(roles) != 0 {
		t.Errorf("expected no roles, got %v", roles)
	}
}

func TestEnforcer_Enforce_CrossDomainDenied(t *testing.T) {
	e := newTestEnforcer(t)

	_, _ = e.AddRoleForUserInDomain("u1", "admin", "d1")
	_, _ = e.AddPolicy("admin", "d1", "*", "*")

	ok, _ := e.Enforce("u1", "d2", "post", "read")
	if ok {
		t.Error("expected cross-domain access to be denied")
	}
}

func TestEnforcer_Enforce_NoRoleNoAccess(t *testing.T) {
	e := newTestEnforcer(t)

	ok, _ := e.Enforce("nobody", "any", "post", "read")
	if ok {
		t.Error("expected user with no role to be denied")
	}
}

func TestEnforcer_AddPolicy_Duplicate(t *testing.T) {
	e := newTestEnforcer(t)

	added1, err := e.AddPolicy("admin", "d1", "*", "*")
	if err != nil {
		t.Fatalf("first add failed: %v", err)
	}
	if !added1 {
		t.Error("expected first add to succeed")
	}

	added2, err := e.AddPolicy("admin", "d1", "*", "*")
	if err != nil {
		t.Fatalf("duplicate add failed: %v", err)
	}
	_ = added2
	// The policy is still present (casbin allows idempotent adds)
	ok, _ := e.Enforce("admin", "d1", "post", "read")
	if !ok {
		t.Error("expected policy to be enforced")
	}
}

func TestEnforcer_RemovePolicy_NotFound(t *testing.T) {
	e := newTestEnforcer(t)

	removed, _ := e.RemovePolicy("nonexistent", "d1", "obj", "act")
	_ = removed
	// Remove of nonexistent policy is a no-op, not an error
}
