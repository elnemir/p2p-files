package operations

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSnapshotRejectsMalformedAndOutOfOrderRecords(t *testing.T) {
	h := snapshotHeader{Version: 1, SchemaHash: strings.Repeat("a", 64), Tables: []string{"first", "second"}}
	raw, e := json.Marshal(h)
	if e != nil {
		t.Fatal(e)
	}
	header := string(raw) + "\n"
	good := header + `{"table":"first","row":{"value":1}}` + "\n" + `{"table":"second","row":{"value":2}}` + "\n"
	accept := func(snapshotHeader) error { return nil }
	if e = scanSnapshot(strings.NewReader(good), accept, nil); e != nil {
		t.Fatal(e)
	}
	bad := []string{
		strings.Replace(good, `"version":1`, `"version":2`, 1),
		strings.Replace(good, `["first","second"]`, `["first","first"]`, 1),
		header + `{"table":"unknown","row":{}}` + "\n",
		header + `{"table":"first","row":null}` + "\n",
		header + `{"table":"second","row":{}}` + "\n" + `{"table":"first","row":{}}` + "\n",
		good + `{"table":"second","row":{}} {"extra":1}`,
		strings.Replace(good, `"version":1`, `"unexpected":true,"version":1`, 1),
	}
	for i, input := range bad {
		if e = scanSnapshot(strings.NewReader(input), accept, nil); e == nil {
			t.Fatalf("accepted malformed snapshot %d", i)
		}
	}
	other := h
	other.SchemaHash = strings.Repeat("b", 64)
	if sameSchema(h, other) {
		t.Fatal("schema mismatch accepted")
	}
}
