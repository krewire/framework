package tui

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestRenderJSON_Deterministic(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
		Port int    `json:"port"`
	}
	a, err := RenderJSON(payload{Name: "x", Port: 1})
	if err != nil {
		t.Fatal(err)
	}
	b, err := RenderJSON(payload{Name: "x", Port: 1})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Error("RenderJSON must be byte-stable across runs")
	}
}

func TestRenderJSON_MapKeysSorted(t *testing.T) {
	out, err := RenderJSON(map[string]int{"b": 2, "a": 1, "c": 3})
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)
	// encoding/json sorts map keys; "a" must precede "b" precedes "c".
	if i, j, k := strings.Index(got, "a"), strings.Index(got, "b"), strings.Index(got, "c"); !(i < j && j < k) {
		t.Errorf("map keys not sorted deterministically: %s", got)
	}
}

func TestResultRoundTrip(t *testing.T) {
	r := ResultOK(map[string]string{"href": "/guide/x"})
	b, err := RenderJSON(r)
	if err != nil {
		t.Fatal(err)
	}
	var back Result
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if !back.OK || back.Data == nil || back.Error != "" {
		t.Errorf("ResultOK round-trip = %+v", back)
	}
	if err := WriteJSON(&bytes.Buffer{}, ResultErr("boom")); err != nil {
		t.Fatal(err)
	}
}
