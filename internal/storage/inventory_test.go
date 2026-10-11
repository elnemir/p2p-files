package storage

import (
	"context"
	"reflect"
	"testing"
)

func TestLocalInventoryStablePagination(t *testing.T) {
	ctx := context.Background()
	p := Local{Root: t.TempDir()}
	for _, key := range []string{"z-last", "a-first", "m-middle"} {
		if e := p.Put(ctx, key, []byte("ciphertext fixture")); e != nil {
			t.Fatal(e)
		}
	}
	keys, next, e := p.List(ctx, "", 2)
	if e != nil || !reflect.DeepEqual(keys, []string{"a-first", "m-middle"}) || next != "m-middle" {
		t.Fatal(keys, next, e)
	}
	keys, next, e = p.List(ctx, next, 2)
	if e != nil || !reflect.DeepEqual(keys, []string{"z-last"}) || next != "" {
		t.Fatal(keys, next, e)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, _, e = p.List(cancelled, "", 2); e == nil {
		t.Fatal("cancelled inventory ran")
	}
	if _, _, e = p.List(ctx, "", 1001); e == nil {
		t.Fatal("unbounded inventory allowed")
	}
}
