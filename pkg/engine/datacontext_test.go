package engine

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

type testProfile struct{ ID string }

func TestDataContext_RegisterPublishGet_Single(t *testing.T) {
	dc := NewDataContext()
	if err := Register[[]string](dc, "config.targets", CardinalitySingle); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if err := Publish(dc, "config.targets", []string{"10.0.0.0/24"}); err != nil {
		t.Fatalf("publish failed: %v", err)
	}
	got, err := Get[[]string](dc, "config.targets")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if len(got) != 1 || got[0] != "10.0.0.0/24" {
		t.Fatalf("unexpected value: %#v", got)
	}
}

func TestDataContext_RegisterAppendGet_List(t *testing.T) {
	dc := NewDataContext()
	// For list, register expected stored type as []testProfile
	if err := Register[[]testProfile](dc, "asset.profiles", CardinalityList); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if err := Append(dc, "asset.profiles", testProfile{ID: "a"}); err != nil {
		t.Fatalf("append failed: %v", err)
	}
	if err := Append(dc, "asset.profiles", testProfile{ID: "b"}); err != nil {
		t.Fatalf("append failed: %v", err)
	}
	got, err := Get[[]testProfile](dc, "asset.profiles")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if len(got) != 2 || got[0].ID != "a" || got[1].ID != "b" {
		t.Fatalf("unexpected list: %#v", got)
	}
}

func TestDataContext_TypeMismatch(t *testing.T) {
	dc := NewDataContext()
	if err := Register[[]string](dc, "config.targets", CardinalitySingle); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	// wrong type
	if err := Publish(dc, "config.targets", "not-a-slice"); err == nil {
		t.Fatalf("expected type mismatch error")
	}
}

func TestRegisterType_NilTypeError(t *testing.T) {
	dc := NewDataContext()
	err := dc.RegisterType("nil.key", nil, CardinalitySingle)
	if err == nil {
		t.Fatal("expected error for nil type")
	}
}

func TestPublishValue_Errors(t *testing.T) {
	dc := NewDataContext()
	// Unregistered key
	err := dc.PublishValue("unreg", "x")
	if err == nil {
		t.Fatal("expected error for unregistered key")
	}

	// Wrong cardinality
	_ = dc.RegisterType("key.list", reflect.TypeOf([]string{}), CardinalityList)
	err = dc.PublishValue("key.list", []string{"a"})
	if err == nil {
		t.Fatal("expected error for wrong cardinality")
	}

	// Type mismatch
	_ = dc.RegisterType("key.single", reflect.TypeOf([]string{}), CardinalitySingle)
	err = dc.PublishValue("key.single", "not-a-slice")
	if err == nil {
		t.Fatal("expected type mismatch error")
	}
}

func TestAppendValue_Errors(t *testing.T) {
	dc := NewDataContext()
	// Unregistered key
	err := dc.AppendValue("missing", "item")
	if err == nil {
		t.Fatal("expected error for unregistered key")
	}

	// Wrong cardinality
	_ = dc.RegisterType("single.key", reflect.TypeOf("string"), CardinalitySingle)
	err = dc.AppendValue("single.key", "item")
	if err == nil {
		t.Fatal("expected error for non-list key")
	}

	// Type mismatch on existing slice
	_ = dc.RegisterType("list.key", reflect.TypeOf([]string{}), CardinalityList)
	dc.data["list.key"] = []int{1}
	err = dc.AppendValue("list.key", "item")
	if err == nil {
		t.Fatal("expected type mismatch for slice element type")
	}
}

func TestGetValue_Errors(t *testing.T) {
	dc := NewDataContext()
	_, err := dc.GetValue("unregistered")
	if err == nil {
		t.Fatal("expected error for unregistered key")
	}

	_ = dc.RegisterType("key", reflect.TypeOf([]string{}), CardinalitySingle)
	_, err = dc.GetValue("key")
	if err == nil {
		t.Fatal("expected error for missing value")
	}

	dc.data["key"] = "wrong-type"
	_, err = dc.GetValue("key")
	if err == nil {
		t.Fatal("expected type mismatch error")
	}
}

func TestCheckTypeLocked_AllPaths(t *testing.T) {
	dc := NewDataContext()
	sch := dataKeySchema{typ: reflect.TypeOf(""), cardinality: CardinalitySingle}

	// nil value with non-interface/pointer should error
	err := dc.checkTypeLocked(sch, nil)
	if err == nil {
		t.Fatal("expected error for nil value with non-pointer type")
	}

	// type mismatch
	err = dc.checkTypeLocked(sch, 123)
	if err == nil {
		t.Fatal("expected type mismatch")
	}

	// correct type
	err = dc.checkTypeLocked(sch, "ok")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGenericHelpers_Get_TypeAssertionFail(t *testing.T) {
	dc := NewDataContext()
	_ = dc.RegisterType("key", reflect.TypeOf([]string{}), CardinalitySingle)
	_ = dc.PublishValue("key", []string{"a"})
	_, err := Get[string](dc, "key")
	if err == nil {
		t.Fatal("expected type assertion failure")
	}
}

func TestAddOrAppendToList_PromoteAndGetAll(t *testing.T) {
	dc := NewDataContext()
	dc.SetInitial("direct.key", "x")
	dc.AddOrAppendToList("direct.key", "y")
	got, ok := dc.Get("direct.key")
	require.True(t, ok)
	require.Equal(t, []interface{}{"x", "y"}, got)
	all := dc.GetAll()
	require.Contains(t, all, "direct.key")
}
