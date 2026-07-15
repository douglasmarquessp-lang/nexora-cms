package cache

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

type testStruct struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func TestNew_MemOnly(t *testing.T) {
	c := New(false)
	if c == nil {
		t.Fatal("expected non-nil cache")
	}
	if c.mem == nil {
		t.Error("expected memory driver")
	}
	if c.redis != nil {
		t.Error("expected no redis driver when memOnly=false")
	}
}

func TestNew_True(t *testing.T) {
	c := New(true)
	if c == nil {
		t.Fatal("expected non-nil cache")
	}
	if c.mem == nil {
		t.Error("expected memory driver")
	}
}

func TestNewWithRedis(t *testing.T) {
	mem := newMemoryCache()
	redis := newMemoryCache()
	c := NewWithRedis(mem, redis)
	if c == nil {
		t.Fatal("expected non-nil cache")
	}
	if c.mem != mem {
		t.Error("memory driver mismatch")
	}
	if c.redis != redis {
		t.Error("redis driver mismatch")
	}
}

func TestCache_Get_Empty(t *testing.T) {
	c := New(false)
	ctx := context.Background()

	val, ok := c.Get(ctx, "nonexistent")
	if ok {
		t.Error("expected false for missing key")
	}
	if val != nil {
		t.Errorf("expected nil, got %v", val)
	}
}

func TestCache_SetAndGet(t *testing.T) {
	c := New(false)
	ctx := context.Background()

	err := c.Set(ctx, "mykey", "myvalue", 5*time.Minute)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	val, ok := c.Get(ctx, "mykey")
	if !ok {
		t.Fatal("expected true for existing key")
	}
	if val != "myvalue" {
		t.Errorf("expected 'myvalue', got %v", val)
	}
}

func TestCache_SetAndGet_Integer(t *testing.T) {
	c := New(false)
	ctx := context.Background()

	err := c.Set(ctx, "intkey", 42, 5*time.Minute)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	val, ok := c.Get(ctx, "intkey")
	if !ok {
		t.Fatal("expected true for existing key")
	}
	// JSON unmarshaling returns float64 for numbers
	f, ok := val.(float64)
	if !ok || f != 42.0 {
		t.Errorf("expected 42.0, got %v (type: %T)", val, val)
	}
}

func TestCache_SetAndGet_Map(t *testing.T) {
	c := New(false)
	ctx := context.Background()

	input := map[string]interface{}{"foo": "bar", "num": 1.0}
	err := c.Set(ctx, "mapkey", input, 5*time.Minute)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	val, ok := c.Get(ctx, "mapkey")
	if !ok {
		t.Fatal("expected true for existing key")
	}
	m, ok := val.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map[string]interface{}, got %T", val)
	}
	if m["foo"] != "bar" {
		t.Errorf("expected 'bar', got %v", m["foo"])
	}
}

func TestCache_Delete(t *testing.T) {
	c := New(false)
	ctx := context.Background()

	c.Set(ctx, "todelete", "value", 5*time.Minute)
	c.Delete(ctx, "todelete")

	val, ok := c.Get(ctx, "todelete")
	if ok {
		t.Error("expected false after delete")
	}
	if val != nil {
		t.Errorf("expected nil, got %v", val)
	}
}

func TestCache_Delete_NonExistent(t *testing.T) {
	c := New(false)
	ctx := context.Background()

	err := c.Delete(ctx, "nonexistent")
	if err != nil {
		t.Errorf("expected no error deleting nonexistent key, got: %v", err)
	}
}

func TestCache_SetJSON_GetJSON(t *testing.T) {
	c := New(false)
	ctx := context.Background()

	input := testStruct{Name: "Test", Age: 30}
	err := c.SetJSON(ctx, "jsonkey", input, 5*time.Minute)
	if err != nil {
		t.Fatalf("SetJSON failed: %v", err)
	}

	var output testStruct
	found, err := c.GetJSON(ctx, "jsonkey", &output)
	if err != nil {
		t.Fatalf("GetJSON failed: %v", err)
	}
	if !found {
		t.Fatal("expected true for existing key")
	}
	if output.Name != "Test" {
		t.Errorf("expected 'Test', got '%s'", output.Name)
	}
	if output.Age != 30 {
		t.Errorf("expected 30, got %d", output.Age)
	}
}

func TestCache_GetJSON_Empty(t *testing.T) {
	c := New(false)
	ctx := context.Background()

	var output testStruct
	found, err := c.GetJSON(ctx, "nonexistent", &output)
	if err != nil {
		t.Fatalf("GetJSON failed: %v", err)
	}
	if found {
		t.Error("expected false for missing key")
	}
}

func TestMemoryCache_Flush(t *testing.T) {
	mc := newMemoryCache()
	ctx := context.Background()

	mc.Set(ctx, "key1", []byte("value1"), 5*time.Minute)
	mc.Set(ctx, "key2", []byte("value2"), 5*time.Minute)
	mc.Flush(ctx)

	val1, _ := mc.Get(ctx, "key1")
	if val1 != nil {
		t.Error("expected nil for key1 after flush")
	}
	val2, _ := mc.Get(ctx, "key2")
	if val2 != nil {
		t.Error("expected nil for key2 after flush")
	}
}

func TestMemoryCache_TTL(t *testing.T) {
	c := New(false)
	ctx := context.Background()

	err := c.Set(ctx, "ttlkey", "expire-me", time.Nanosecond)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	time.Sleep(5 * time.Millisecond)

	val, ok := c.Get(ctx, "ttlkey")
	if ok {
		t.Error("expected key to have expired")
	}
	if val != nil {
		t.Errorf("expected nil, got %v", val)
	}
}

func TestMemoryCache_TTL_NotExpired(t *testing.T) {
	c := New(false)
	ctx := context.Background()

	err := c.Set(ctx, "noexpire", "persist", 5*time.Minute)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	val, ok := c.Get(ctx, "noexpire")
	if !ok {
		t.Fatal("expected true for unexpired key")
	}
	if val != "persist" {
		t.Errorf("expected 'persist', got %v", val)
	}
}

func TestMemoryCache_TTL_Zero(t *testing.T) {
	c := New(false)
	ctx := context.Background()

	err := c.Set(ctx, "zero", "no-ttl", 0)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	time.Sleep(5 * time.Millisecond)

	val, ok := c.Get(ctx, "zero")
	if !ok {
		t.Fatal("expected true for zero-TTL key (never expires)")
	}
	if val != "no-ttl" {
		t.Errorf("expected 'no-ttl', got %v", val)
	}
}

func TestCache_MultipleKeys(t *testing.T) {
	c := New(false)
	ctx := context.Background()

	c.Set(ctx, "a", 1, 5*time.Minute)
	c.Set(ctx, "b", 2, 5*time.Minute)
	c.Set(ctx, "c", 3, 5*time.Minute)

	va, oka := c.Get(ctx, "a")
	vb, okb := c.Get(ctx, "b")
	vc, okc := c.Get(ctx, "c")

	if !oka || !okb || !okc {
		t.Error("expected all keys to be found")
	}
	fa, _ := va.(float64)
	fb, _ := vb.(float64)
	fc, _ := vc.(float64)
	if fa != 1.0 || fb != 2.0 || fc != 3.0 {
		t.Errorf("unexpected values: %v, %v, %v", va, vb, vc)
	}
}

func TestCache_Overwrite(t *testing.T) {
	c := New(false)
	ctx := context.Background()

	c.Set(ctx, "key", "old", 5*time.Minute)
	c.Set(ctx, "key", "new", 5*time.Minute)

	val, ok := c.Get(ctx, "key")
	if !ok {
		t.Fatal("expected true for overwritten key")
	}
	if val != "new" {
		t.Errorf("expected 'new', got %v", val)
	}
}

func TestNew_ClearState(t *testing.T) {
	c1 := New(false)
	ctx := context.Background()

	c1.Set(ctx, "shared", "value", 5*time.Minute)

	c2 := New(false)
	_, ok := c2.Get(ctx, "shared")
	if ok {
		t.Error("expected new cache to be empty")
	}
}

func TestCache_Get_MalformedJSON(t *testing.T) {
	c := New(false)
	ctx := context.Background()

	c.mem.Set(ctx, "badjson", []byte(`{invalid json`), 5*time.Minute)

	val, ok := c.Get(ctx, "badjson")
	if !ok {
		t.Fatal("expected true for malformed JSON (returns raw bytes)")
	}
	data, ok := val.([]byte)
	if !ok {
		t.Fatalf("expected []byte, got %T", val)
	}
	if string(data) != `{invalid json` {
		t.Errorf("expected '{invalid json', got '%s'", string(data))
	}
}

func TestCache_Get_MalformedJSON_FromMem(t *testing.T) {
	c := New(false)
	ctx := context.Background()

	c.mem.Set(ctx, "badjson2", []byte(`not json at all`), 5*time.Minute)

	val, ok := c.Get(ctx, "badjson2")
	if !ok {
		t.Fatal("expected true for malformed JSON")
	}
	data, ok := val.([]byte)
	if !ok {
		t.Fatalf("expected []byte, got %T", val)
	}
	if string(data) != "not json at all" {
		t.Errorf("expected 'not json at all', got '%s'", string(data))
	}
}

func TestCache_Set_NilValue(t *testing.T) {
	c := New(false)
	ctx := context.Background()

	err := c.Set(ctx, "nilkey", nil, 5*time.Minute)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	val, ok := c.Get(ctx, "nilkey")
	if !ok {
		t.Fatal("expected true for nil value key")
	}
	if val != nil {
		t.Errorf("expected nil, got %v", val)
	}
}

func TestCache_Set_StructAsValue(t *testing.T) {
	c := New(false)
	ctx := context.Background()

	type person struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	input := person{Name: "Bob", Age: 30}
	err := c.Set(ctx, "person", input, 5*time.Minute)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	val, ok := c.Get(ctx, "person")
	if !ok {
		t.Fatal("expected true for struct key")
	}
	m, ok := val.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map[string]interface{}, got %T", val)
	}
	if m["name"] != "Bob" {
		t.Errorf("expected 'Bob', got %v", m["name"])
	}
	if m["age"] != float64(30) {
		t.Errorf("expected 30, got %v", m["age"])
	}
}

func TestCache_Set_BoolValue(t *testing.T) {
	c := New(false)
	ctx := context.Background()

	err := c.Set(ctx, "boolkey", true, 5*time.Minute)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	val, ok := c.Get(ctx, "boolkey")
	if !ok {
		t.Fatal("expected true for bool key")
	}
	b, ok := val.(bool)
	if !ok {
		t.Fatalf("expected bool, got %T", val)
	}
	if !b {
		t.Error("expected true")
	}
}

func TestCache_Set_ArrayValue(t *testing.T) {
	c := New(false)
	ctx := context.Background()

	input := []string{"a", "b", "c"}
	err := c.Set(ctx, "arrkey", input, 5*time.Minute)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	val, ok := c.Get(ctx, "arrkey")
	if !ok {
		t.Fatal("expected true for array key")
	}
	arr, ok := val.([]interface{})
	if !ok {
		t.Fatalf("expected []interface{}, got %T", val)
	}
	if len(arr) != 3 {
		t.Errorf("expected 3 elements, got %d", len(arr))
	}
	if arr[0] != "a" || arr[1] != "b" || arr[2] != "c" {
		t.Errorf("unexpected array values: %v", arr)
	}
}

func TestCache_ConcurrentAccess(t *testing.T) {
	c := New(false)
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := fmt.Sprintf("concurrent-%d", n)
			err := c.Set(ctx, key, n, 5*time.Minute)
			if err != nil {
				t.Errorf("Set failed for %s: %v", key, err)
				return
			}
			val, ok := c.Get(ctx, key)
			if !ok {
				t.Errorf("Get failed for %s", key)
				return
			}
			f, ok := val.(float64)
			if !ok || int(f) != n {
				t.Errorf("unexpected value for %s: %v (type: %T)", key, val, val)
			}
		}(i)
	}
	wg.Wait()
}

func TestCache_ConcurrentOverwrite(t *testing.T) {
	c := New(false)
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_ = c.Set(ctx, "shared", n, 5*time.Minute)
		}(i)
	}
	wg.Wait()

	_, ok := c.Get(ctx, "shared")
	if !ok {
		t.Error("expected shared key to exist after concurrent writes")
	}
}

func TestCache_LargeValue(t *testing.T) {
	c := New(false)
	ctx := context.Background()

	large := make([]int, 10000)
	for i := range large {
		large[i] = i
	}

	err := c.Set(ctx, "large", large, 5*time.Minute)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	val, ok := c.Get(ctx, "large")
	if !ok {
		t.Fatal("expected true for large value")
	}
	arr, ok := val.([]interface{})
	if !ok {
		t.Fatalf("expected []interface{}, got %T", val)
	}
	if len(arr) != 10000 {
		t.Errorf("expected 10000 elements, got %d", len(arr))
	}
	for i := 0; i < 10; i++ {
		if arr[i].(float64) != float64(i) {
			t.Errorf("arr[%d] = %v, expected %d", i, arr[i], i)
		}
	}
}

func TestMemoryCache_TTL_AccessAfterExpiry(t *testing.T) {
	c := New(false)
	ctx := context.Background()

	err := c.Set(ctx, "expire-fast", "gone", time.Nanosecond)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	time.Sleep(5 * time.Millisecond)

	val, ok := c.Get(ctx, "expire-fast")
	if ok {
		t.Error("expected key to have expired")
	}
	if val != nil {
		t.Errorf("expected nil, got %v", val)
	}

	exists, _ := c.mem.Exists(ctx, "expire-fast")
	if exists {
		t.Error("expected expired entry to be cleaned up from store")
	}
}

func TestMemoryCache_TTL_PartialExpiry(t *testing.T) {
	c := New(false)
	ctx := context.Background()

	c.Set(ctx, "permanent", "stay", 5*time.Minute)
	c.Set(ctx, "temporary", "go", time.Nanosecond)

	time.Sleep(5 * time.Millisecond)

	val1, ok1 := c.Get(ctx, "permanent")
	if !ok1 {
		t.Error("expected permanent key to exist")
	} else if val1 != "stay" {
		t.Errorf("expected 'stay', got %v", val1)
	}

	val2, ok2 := c.Get(ctx, "temporary")
	if ok2 {
		t.Error("expected temporary key to have expired")
	}
	if val2 != nil {
		t.Errorf("expected nil, got %v", val2)
	}
}

func TestMemoryCache_Exists(t *testing.T) {
	mc := newMemoryCache()
	ctx := context.Background()

	exists, err := mc.Exists(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if exists {
		t.Error("expected false for nonexistent key")
	}

	mc.Set(ctx, "existent", []byte("value"), 5*time.Minute)
	exists, err = mc.Exists(ctx, "existent")
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !exists {
		t.Error("expected true for existing key")
	}
}

func TestNewWithRedis_NilMem(t *testing.T) {
	mem := newMemoryCache()
	redis := newMemoryCache()
	c := NewWithRedis(mem, redis)
	if c.redis != redis {
		t.Error("redis driver mismatch")
	}
	if c.mem != mem {
		t.Error("memory driver mismatch")
	}
}

func TestCache_GetJSON_NonExistent(t *testing.T) {
	c := New(false)
	ctx := context.Background()

	var dest testStruct
	found, err := c.GetJSON(ctx, "no-such-key", &dest)
	if err != nil {
		t.Fatalf("GetJSON failed: %v", err)
	}
	if found {
		t.Error("expected false for nonexistent key")
	}
}

func TestNewWithRedis_Get_RedisHit(t *testing.T) {
	mem := newMemoryCache()
	redis := newMemoryCache()
	c := NewWithRedis(mem, redis)
	ctx := context.Background()

	redis.Set(ctx, "rediskey", []byte(`"from-redis"`), 5*time.Minute)

	val, ok := c.Get(ctx, "rediskey")
	if !ok {
		t.Fatal("expected true for key in redis")
	}
	if val != "from-redis" {
		t.Errorf("expected 'from-redis', got %v", val)
	}
}

func TestNewWithRedis_Get_RedisMiss_MemHit(t *testing.T) {
	mem := newMemoryCache()
	redis := newMemoryCache()
	c := NewWithRedis(mem, redis)
	ctx := context.Background()

	mem.Set(ctx, "memkey", []byte(`"from-mem"`), 5*time.Minute)

	val, ok := c.Get(ctx, "memkey")
	if !ok {
		t.Fatal("expected true for key in mem")
	}
	if val != "from-mem" {
		t.Errorf("expected 'from-mem', got %v", val)
	}
}

func TestNewWithRedis_Set_StoresBoth(t *testing.T) {
	mem := newMemoryCache()
	redis := newMemoryCache()
	c := NewWithRedis(mem, redis)
	ctx := context.Background()

	err := c.Set(ctx, "both", "shared-value", 5*time.Minute)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	memData, _ := mem.Get(ctx, "both")
	if memData == nil {
		t.Error("expected value in mem")
	}

	redisData, _ := redis.Get(ctx, "both")
	if redisData == nil {
		t.Error("expected value in redis")
	}
}

func TestNewWithRedis_Delete_RemovesBoth(t *testing.T) {
	mem := newMemoryCache()
	redis := newMemoryCache()
	c := NewWithRedis(mem, redis)
	ctx := context.Background()

	c.Set(ctx, "delboth", "value", 5*time.Minute)
	c.Delete(ctx, "delboth")

	memData, _ := mem.Get(ctx, "delboth")
	if memData != nil {
		t.Error("expected nil in mem after delete")
	}

	redisData, _ := redis.Get(ctx, "delboth")
	if redisData != nil {
		t.Error("expected nil in redis after delete")
	}
}

func TestNewWithRedis_GetJSON_RedisHit(t *testing.T) {
	mem := newMemoryCache()
	redis := newMemoryCache()
	c := NewWithRedis(mem, redis)
	ctx := context.Background()

	redis.Set(ctx, "jsonkey", []byte(`{"name":"Redis","age":10}`), 5*time.Minute)

	var dest testStruct
	found, err := c.GetJSON(ctx, "jsonkey", &dest)
	if err != nil {
		t.Fatalf("GetJSON failed: %v", err)
	}
	if !found {
		t.Fatal("expected true for key in redis")
	}
	if dest.Name != "Redis" {
		t.Errorf("expected 'Redis', got '%s'", dest.Name)
	}
}

func TestNewWithRedis_GetJSON_RedisMiss_MemHit(t *testing.T) {
	mem := newMemoryCache()
	redis := newMemoryCache()
	c := NewWithRedis(mem, redis)
	ctx := context.Background()

	mem.Set(ctx, "memjson", []byte(`{"name":"Mem","age":20}`), 5*time.Minute)

	var dest testStruct
	found, err := c.GetJSON(ctx, "memjson", &dest)
	if err != nil {
		t.Fatalf("GetJSON failed: %v", err)
	}
	if !found {
		t.Fatal("expected true for key in mem")
	}
	if dest.Name != "Mem" {
		t.Errorf("expected 'Mem', got '%s'", dest.Name)
	}
}

func TestNewWithRedis_SetJSON_StoresBoth(t *testing.T) {
	mem := newMemoryCache()
	redis := newMemoryCache()
	c := NewWithRedis(mem, redis)
	ctx := context.Background()

	input := testStruct{Name: "Both", Age: 99}
	err := c.SetJSON(ctx, "bothjson", input, 5*time.Minute)
	if err != nil {
		t.Fatalf("SetJSON failed: %v", err)
	}

	memData, _ := mem.Get(ctx, "bothjson")
	if memData == nil {
		t.Error("expected value in mem")
	}

	redisData, _ := redis.Get(ctx, "bothjson")
	if redisData == nil {
		t.Error("expected value in redis")
	}
}

func TestNewWithRedis_Get_MalformedJSONInRedis(t *testing.T) {
	mem := newMemoryCache()
	redis := newMemoryCache()
	c := NewWithRedis(mem, redis)
	ctx := context.Background()

	redis.Set(ctx, "bad", []byte(`not json`), 5*time.Minute)

	val, ok := c.Get(ctx, "bad")
	if !ok {
		t.Fatal("expected true for malformed JSON (returns raw bytes)")
	}
	data, ok := val.([]byte)
	if !ok {
		t.Fatalf("expected []byte, got %T", val)
	}
	if string(data) != "not json" {
		t.Errorf("expected 'not json', got '%s'", string(data))
	}
}

func TestCache_Set_ZeroTTL(t *testing.T) {
	c := New(false)
	ctx := context.Background()

	err := c.Set(ctx, "zero-ttl", "value", 0)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	val, ok := c.Get(ctx, "zero-ttl")
	if !ok {
		t.Fatal("expected true for zero-TTL key")
	}
	if val != "value" {
		t.Errorf("expected 'value', got %v", val)
	}
}
