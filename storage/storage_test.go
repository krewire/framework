// Tests for KWF-AST-K7Q2M
package storage

import (
	"context"
	"testing"

	"github.com/krewire/framework/app"
	ftest "github.com/krewire/framework/test"
)

// kvBackends returns every backend so behavior tests run against the contract.
func kvBackends(t *testing.T) map[string]KV {
	t.Helper()
	file, err := NewFile(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return map[string]KV{
		"memory": NewMemory(),
		"file":   file,
	}
}

// Spec: KWF-AST-K7Q2M FRK-AST-010 Scope: Unit
func TestFRK_AST_010_KVContract_AllBackends(t *testing.T) {
	ctx := context.Background()
	for name, kv := range kvBackends(t) {
		t.Run(name, func(t *testing.T) {
			_, ok, err := kv.Get(ctx, "missing")
			ftest.NoError(t, err)
			if ok {
				t.Error("missing key reported ok")
			}

			ftest.NoError(t, kv.Put(ctx, "a/b.txt", []byte("hello")))
			val, ok, err := kv.Get(ctx, "a/b.txt")
			ftest.NoError(t, err)
			if !ok || string(val) != "hello" {
				t.Errorf("get = %q %v", val, ok)
			}

			keys, err := kv.List(ctx, "a/")
			ftest.NoError(t, err)
			ftest.Equal(t, 1, len(keys))
			if len(keys) == 1 {
				ftest.Equal(t, "a/b.txt", keys[0])
			}

			ftest.NoError(t, kv.Put(ctx, "a/c.txt", []byte("x")))
			all, err := kv.List(ctx, "")
			ftest.NoError(t, err)
			ftest.Equal(t, 2, len(all))

			ftest.NoError(t, kv.Delete(ctx, "a/b.txt"))
			_, ok, err = kv.Get(ctx, "a/b.txt")
			ftest.NoError(t, err)
			if ok {
				t.Error("deleted key still present")
			}
			// deleting absent key is not an error
			ftest.NoError(t, kv.Delete(ctx, "a/b.txt"))
		})
	}
}

// Spec: KWF-AST-K7Q2M FRK-AST-012 Scope: Unit
func TestFRK_AST_012_FileKV_RejectsTraversal(t *testing.T) {
	kv, err := NewFile(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	err = kv.Put(context.Background(), "../escape.txt", []byte("x"))
	if err == nil {
		t.Fatal("expected traversal rejection")
	}
}

// Spec: KWF-AST-K7Q2M FRK-AST-013 Scope: Unit
func TestFRK_AST_013_Provider_BindsIntoContainer(t *testing.T) {
	c, err := app.NewApp(Provider(NewMemory())).Build()
	if err != nil {
		t.Fatal(err)
	}
	got, err := app.Resolve[*KV](c)
	if err != nil {
		t.Fatal(err)
	}
	kv := *got
	if err := kv.Put(context.Background(), "k", []byte("v")); err != nil {
		t.Fatal(err)
	}
	val, ok, err := kv.Get(context.Background(), "k")
	if err != nil || !ok || string(val) != "v" {
		t.Errorf("resolved kv broken: %q %v %v", val, ok, err)
	}
}
