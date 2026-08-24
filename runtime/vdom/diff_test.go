// Tests for KWF-T4X9P
package vdom

import (
	"strings"
	"testing"
)

// Spec: KWF-T4X9P FRK-WASM-020 Scope: Package
func TestFRK_WASM_020_VNode_HoldsTagPropsChildrenKeyAndComponentDiscriminator(t *testing.T) {
	el := El("li", map[string]string{"class": "item"}, Text("x"))
	if el.Kind != KindElement || el.Tag != "li" || el.Props["class"] != "item" {
		t.Fatalf("element fields = %+v", el)
	}
	if len(el.Children) != 1 || el.Children[0].Kind != KindText {
		t.Fatalf("children not retained: %+v", el.Children)
	}

	txt := Text("hello")
	if txt.Kind != KindText || txt.Text != "hello" {
		t.Fatalf("text fields = %+v", txt)
	}

	comp := Component("Counter", "c1", map[string]string{"start": "0"})
	if comp.Kind != KindComponent || comp.ComponentName != "Counter" ||
		comp.Key != "c1" || comp.Props["start"] != "0" {
		t.Fatalf("component discriminator incomplete: %+v", comp)
	}
}

// Spec: KWF-T4X9P FRK-WASM-021 Scope: Package
func TestFRK_WASM_021_Diff_KeyedReorder_EmitsRemovesThenInserts(t *testing.T) {
	old := El("ul", nil,
		keyed("a"), keyed("b"), keyed("c"),
	)
	next := El("ul", nil,
		keyed("c"), El("li", nil, Text("B")), keyed("a"),
	)

	patches := Diff(old, next)
	if len(patches) != 4 {
		t.Fatalf("want 4 patches, got %d: %+v", len(patches), patches)
	}
	want := []struct {
		kind  PatchKind
		index int
	}{
		{PatchRemove, 1}, {PatchRemove, 0}, {PatchInsert, 1}, {PatchInsert, 2},
	}
	for i, w := range want {
		p := patches[i]
		if p.Kind != w.kind || p.Index != w.index {
			t.Fatalf("patch[%d] = kind %d index %d, want kind %d index %d (all: %+v)", i, p.Kind, p.Index, w.kind, w.index, patches)
		}
	}
}

// Spec: KWF-T4X9P FRK-WASM-021 Scope: Package
func TestFRK_WASM_021_Diff_NestedKeyedChild_RecursesWithPath(t *testing.T) {
	old := El("ul", nil, wrap("a", "old"))
	next := El("ul", nil, wrap("a", "new"))

	patches := Diff(old, next)
	if len(patches) != 1 || patches[0].Kind != PatchUpdateText {
		t.Fatalf("want single nested UpdateText, got %+v", patches)
	}
	if got := patches[0].Path; len(got) != 1 || got[0] != 0 {
		t.Fatalf("path through keyed child = %v, want [0]", got)
	}
	if patches[0].Text != "new" {
		t.Fatalf("text = %q", patches[0].Text)
	}
}

// Spec: KWF-T4X9P FRK-WASM-021 Scope: Package
func TestFRK_WASM_021_Diff_UnkeyedShrink_PairsPositionallyAndTrimsTail(t *testing.T) {
	old := El("div", nil, Text("1"), Text("2"), Text("3"))
	next := El("div", nil, Text("1"), Text("9"))

	patches := Diff(old, next)
	counts := map[PatchKind]int{}
	for _, p := range patches {
		counts[p.Kind]++
	}
	if counts[PatchUpdateText] != 1 || counts[PatchRemove] != 1 || counts[PatchInsert] != 0 {
		t.Fatalf("shrink reconcile mismatch: %+v", patches)
	}
}

// Spec: KWF-T4X9P FRK-WASM-021 Scope: Package
func TestFRK_WASM_021_Diff_UnkeyedGrow_AppendsInsert(t *testing.T) {
	old := El("div", nil, Text("1"))
	next := El("div", nil, Text("1"), Text("2"))

	patches := Diff(old, next)
	if len(patches) != 1 || patches[0].Kind != PatchInsert || patches[0].Index != 1 {
		t.Fatalf("grow should append one insert at new index 1: %+v", patches)
	}
}

// Spec: KWF-T4X9P FRK-WASM-021 Scope: Package
func TestFRK_WASM_021_Diff_MixedKeys_UnkeyedConsumesLeftoverSlots(t *testing.T) {
	old := El("div", nil, keyed("k"), Text("free"))
	next := El("div", nil, Text("moved"), keyed("k"))

	patches := Diff(old, next)
	for _, p := range patches {
		if p.Kind == PatchInsert || p.Kind == PatchRemove {
			t.Fatalf("stable mixed list needs only content patches, got %+v", patches)
		}
	}
}

// Spec: KWF-T4X9P FRK-WASM-021 Scope: Package
func TestFRK_WASM_021_Diff_TagOrKindChange_Replaces(t *testing.T) {
	tagSwap := Diff(El("span", nil), El("div", nil))
	if len(tagSwap) != 1 || tagSwap[0].Kind != PatchReplace {
		t.Fatalf("tag change should replace: %+v", tagSwap)
	}

	textToEl := Diff(Text("x"), El("b", nil))
	if len(textToEl) != 1 || textToEl[0].Kind != PatchReplace {
		t.Fatalf("kind change should replace: %+v", textToEl)
	}

	nilOut := Diff(El("p", nil), nil)
	if len(nilOut) != 1 || nilOut[0].Kind != PatchRemove {
		t.Fatalf("nil next should remove root: %+v", nilOut)
	}

	nilIn := Diff(nil, El("p", nil))
	if len(nilIn) != 1 || nilIn[0].Kind != PatchInsert {
		t.Fatalf("nil old should insert root: %+v", nilIn)
	}

	if both := Diff(nil, nil); both != nil {
		t.Fatalf("nil vs nil should be no-op, got %+v", both)
	}
}

// Spec: KWF-T4X9P FRK-WASM-021 Scope: Package
func TestFRK_WASM_021_Diff_PropsChanged_EmitsSingleUpdateProps(t *testing.T) {
	old := El("input", map[string]string{"type": "text"})
	next := El("input", map[string]string{"type": "password"})

	patches := Diff(old, next)
	if len(patches) != 1 || patches[0].Kind != PatchUpdateProps {
		t.Fatalf("want one UpdateProps, got %+v", patches)
	}
	if patches[0].Props["type"] != "password" {
		t.Fatalf("patch props = %+v", patches[0].Props)
	}

	same := Diff(El("i", map[string]string{"id": "k"}), El("i", map[string]string{"id": "k"}))
	if len(same) != 0 {
		t.Fatalf("identical props must yield no patch: %+v", same)
	}
}

// Spec: KWF-T4X9P FRK-WASM-023 Scope: Package
func TestFRK_WASM_023_Diff_CoversComponentRenameAndNilSiblings(t *testing.T) {
	rename := Diff(Component("Alpha", "", nil), Component("Beta", "", nil))
	if len(rename) != 1 || rename[0].Kind != PatchReplace {
		t.Fatalf("component rename should replace: %+v", rename)
	}

	withNils := El("ul", nil, nil, keyed("x"), nil)
	clean := El("ul", nil, keyed("x"))
	if patches := Diff(withNils, clean); len(patches) != 0 {
		t.Fatalf("nil siblings must be invisible to diff: %+v", patches)
	}
}

func keyed(k string) *VNode {
	return &VNode{Kind: KindElement, Tag: "li", Key: k, Children: []*VNode{Text(strings.ToUpper(k))}}
}

func wrap(key, s string) *VNode {
	return &VNode{Kind: KindElement, Tag: "li", Key: key, Children: []*VNode{Text(s)}}
}
