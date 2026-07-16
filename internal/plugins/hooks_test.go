package plugins

import (
	"context"
	"errors"
	"testing"
)

func TestHooks_AddAction(t *testing.T) {
	h := NewHooks()
	ctx := context.Background()
	called := false

	h.AddAction("test-p", HookAfterPostSave, func(ctx context.Context, args map[string]interface{}) error {
		called = true
		return nil
	}, 10)

	if err := h.DoAction(ctx, HookAfterPostSave, nil); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Error("action not called")
	}
}

func TestHooks_ActionPriority(t *testing.T) {
	h := NewHooks()
	ctx := context.Background()
	var order []int

	h.AddAction("p1", HookBeforeRender, func(ctx context.Context, args map[string]interface{}) error {
		order = append(order, 1)
		return nil
	}, 10)

	h.AddAction("p2", HookBeforeRender, func(ctx context.Context, args map[string]interface{}) error {
		order = append(order, 2)
		return nil
	}, 5)

	h.DoAction(ctx, HookBeforeRender, nil)
	if len(order) != 2 || order[0] != 2 || order[1] != 1 {
		t.Errorf("order = %v, want [2,1]", order)
	}
}

func TestHooks_ActionError(t *testing.T) {
	h := NewHooks()
	ctx := context.Background()
	expectedErr := errors.New("action failed")

	h.AddAction("test-p", HookBeforePostSave, func(ctx context.Context, args map[string]interface{}) error {
		return expectedErr
	}, 10)

	err := h.DoAction(ctx, HookBeforePostSave, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestHooks_NoAction(t *testing.T) {
	h := NewHooks()
	ctx := context.Background()
	if err := h.DoAction(ctx, HookBeforeLogin, nil); err != nil {
		t.Fatal(err)
	}
}

func TestHooks_AddFilter(t *testing.T) {
	h := NewHooks()
	ctx := context.Background()

	h.AddFilter("test-p", HookBeforeRender, func(ctx context.Context, value interface{}, args map[string]interface{}) (interface{}, error) {
		s := value.(string)
		return s + " filtered", nil
	}, 10)

	result, err := h.ApplyFilter(ctx, HookBeforeRender, "hello", nil)
	if err != nil {
		t.Fatal(err)
	}
	if result != "hello filtered" {
		t.Errorf("result = %q", result)
	}
}

func TestHooks_FilterPriority(t *testing.T) {
	h := NewHooks()
	ctx := context.Background()

	h.AddFilter("p1", HookBeforeRender, func(ctx context.Context, value interface{}, args map[string]interface{}) (interface{}, error) {
		return value.(string) + " second", nil
	}, 10)

	h.AddFilter("p2", HookBeforeRender, func(ctx context.Context, value interface{}, args map[string]interface{}) (interface{}, error) {
		return value.(string) + " first", nil
	}, 5)

	result, _ := h.ApplyFilter(ctx, HookBeforeRender, "", nil)
	if result != " first second" {
		t.Errorf("result = %q", result)
	}
}

func TestHooks_FilterError(t *testing.T) {
	h := NewHooks()
	ctx := context.Background()

	h.AddFilter("test-p", HookAfterRender, func(ctx context.Context, value interface{}, args map[string]interface{}) (interface{}, error) {
		return nil, errors.New("filter failed")
	}, 10)

	_, err := h.ApplyFilter(ctx, HookAfterRender, "val", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestHooks_NoFilter(t *testing.T) {
	h := NewHooks()
	ctx := context.Background()
	result, err := h.ApplyFilter(ctx, HookAfterLogin, "val", nil)
	if err != nil {
		t.Fatal(err)
	}
	if result != "val" {
		t.Errorf("result = %q", result)
	}
}

func TestHooks_RemovePlugin(t *testing.T) {
	h := NewHooks()
	ctx := context.Background()

	h.AddAction("p1", HookAfterPostSave, func(ctx context.Context, args map[string]interface{}) error {
		return nil
	}, 10)
	h.AddFilter("p1", HookBeforeRender, func(ctx context.Context, value interface{}, args map[string]interface{}) (interface{}, error) {
		return value, nil
	}, 10)

	h.RemovePlugin("p1")
	if err := h.DoAction(ctx, HookAfterPostSave, nil); err != nil {
		t.Fatal(err)
	}

	h.AddAction("p2", HookAfterPostSave, func(ctx context.Context, args map[string]interface{}) error {
		return nil
	}, 10)

	regs := h.GetRegistrations("p2")
	if len(regs) != 1 {
		t.Errorf("registrations = %d", len(regs))
	}
}

func TestIsValidHook(t *testing.T) {
	if !IsValidHook(string(HookAfterPostSave)) {
		t.Error("HookAfterPostSave should be valid")
	}
	if IsValidHook("invalid_hook") {
		t.Error("invalid_hook should not be valid")
	}
}

func TestAvailableHooks(t *testing.T) {
	if len(AvailableHooks) == 0 {
		t.Fatal("AvailableHooks is empty")
	}
}
