package vdom

import "sort"

// PatchKind enumerates the reconciliation operations emitted by Diff.
type PatchKind uint8

const (
	// PatchInsert adds Node into the parent at Index, interpreted in the NEW
	// child layout.
	PatchInsert PatchKind = iota
	// PatchRemove deletes the old child at Index, interpreted in the OLD
	// child layout.
	PatchRemove
	// PatchUpdateProps replaces the element's props with Props
	// (replace-all semantics over normalized props).
	PatchUpdateProps
	// PatchUpdateText sets the text node's content to Text.
	PatchUpdateText
	// PatchReplace swaps the node at Path (root level: nil Path) with Node.
	PatchReplace
)

// Patch is one reconciliation operation. Path holds the child indices from
// the diffed root down to the affected parent. Removes are emitted first in
// descending OLD-layout order, followed by property/text updates and inserts
// in ascending NEW-layout order, so applying the slice front-to-back is safe.
type Patch struct {
	Kind  PatchKind
	Path  []int
	Index int
	Key   string

	Node  *VNode
	Props map[string]string
	Text  string
}

// Diff reconciles old into next and returns patches that transform a rendered
// old tree into next. Nil arguments are tolerated: diffing nil against a tree
// yields a single insert and vice versa a single remove.
//
// Children are reconciled positionally when neither side uses keys. When keys
// are present, keyed children are matched by key in O(n); a keyed child whose
// position would move is decomposed — per this spec's fixed patch vocabulary —
// into a remove at its old position plus an insert carrying the fresh subtree
// at its new position. Unkeyed children inside a keyed list consume leftover
// old slots in order.
func Diff(old, next *VNode) []Patch {
	var patches []Patch
	diffNode(nil, old, next, &patches)
	return patches
}

func diffNode(path []int, old, next *VNode, out *[]Patch) {
	var parent []int
	own := 0
	if len(path) > 0 {
		parent = path[:len(path)-1]
		own = path[len(path)-1]
	}

	switch {
	case old == nil && next == nil:
		return
	case old == nil:
		emit(out, Patch{Kind: PatchInsert, Node: next, Key: next.Key})
		return
	case next == nil:
		emit(out, Patch{Kind: PatchRemove})
		return
	case old.Kind != next.Kind || displayTag(old) != displayTag(next):
		emit(out, Patch{Kind: PatchReplace, Path: clonePath(parent), Index: own, Node: next, Key: next.Key})
		return
	}

	if old.Kind == KindText {
		if old.Text != next.Text {
			emit(out, Patch{Kind: PatchUpdateText, Path: clonePath(parent), Index: own, Text: next.Text})
		}
		return
	}

	oldProps, nextProps := NormalizeProps(old.Props), NormalizeProps(next.Props)
	if !propsEqual(oldProps, nextProps) {
		emit(out, Patch{Kind: PatchUpdateProps, Path: clonePath(parent), Index: own, Props: nextProps})
	}

	diffChildren(clonePath(path), old.Children, next.Children, out)
}

func diffChildren(path []int, old, next []*VNode, out *[]Patch) {
	old, next = dropNil(old), dropNil(next)

	nextKeys := make(map[string]bool, len(next))
	for _, n := range next {
		if n.Key != "" {
			nextKeys[n.Key] = true
		}
	}
	nextHasKey := len(nextKeys) > 0

	oldByKey := make(map[string]int, len(old))
	for i, o := range old {
		if o.Key != "" {
			if _, dup := oldByKey[o.Key]; !dup {
				oldByKey[o.Key] = i
			}
		}
	}

	matchNewToOld := make(map[int]int, len(old))
	for i, n := range next {
		if n.Key == "" {
			continue
		}
		if oi, ok := oldByKey[n.Key]; ok {
			matchNewToOld[i] = oi
		}
	}

	var removes []Patch
	initiallyMatched := make([]bool, len(old))
	for _, oi := range matchNewToOld {
		initiallyMatched[oi] = true
	}
	lastMatched := -1
	for ni := 0; ni < len(next); ni++ {
		oi, ok := matchNewToOld[ni]
		if !ok {
			continue
		}
		if oi < lastMatched {
			delete(matchNewToOld, ni)
			removes = append(removes, Patch{Kind: PatchRemove, Path: path, Index: oi})
		} else {
			lastMatched = oi
		}
	}
	for i, o := range old {
		if o.Key != "" && !initiallyMatched[i] {
			removes = append(removes, Patch{Kind: PatchRemove, Path: path, Index: i})
		}
	}

	var updates, inserts []Patch
	freeOld := make([]int, 0, len(old))
	if nextHasKey {
		for i, o := range old {
			if o.Key == "" {
				freeOld = append(freeOld, i)
			}
		}
	}
	freePos := 0

	for i, n := range next {
		switch {
		case n.Key != "":
			if oi, ok := matchNewToOld[i]; ok {
				diffNode(append(path, i), old[oi], n, &updates)
			} else {
				inserts = append(inserts, Patch{Kind: PatchInsert, Path: path, Index: i, Node: n, Key: n.Key})
			}
		case !nextHasKey:
			if i < len(old) {
				diffNode(append(path, i), old[i], n, &updates)
			} else {
				inserts = append(inserts, Patch{Kind: PatchInsert, Path: path, Index: i, Node: n})
			}
		default:
			paired := false
			for freePos < len(freeOld) {
				idx := freeOld[freePos]
				freePos++
				diffNode(append(path, i), old[idx], n, &updates)
				paired = true
				break
			}
			if !paired {
				inserts = append(inserts, Patch{Kind: PatchInsert, Path: path, Index: i, Node: n})
			}
		}
	}
	if !nextHasKey {
		for i := len(next); i < len(old); i++ {
			removes = append(removes, Patch{Kind: PatchRemove, Path: path, Index: i})
		}
	}

	sort.SliceStable(removes, func(a, b int) bool { return removes[a].Index > removes[b].Index })
	*out = append(*out, removes...)
	*out = append(*out, updates...)
	*out = append(*out, inserts...)
}

func emit(out *[]Patch, p Patch) { *out = append(*out, p) }

func clonePath(path []int) []int {
	if len(path) == 0 {
		return nil
	}
	cp := make([]int, len(path), len(path)+1)
	copy(cp, path)
	return cp
}

func dropNil(nodes []*VNode) []*VNode {
	out := nodes[:0:0]
	for _, n := range nodes {
		if n != nil {
			out = append(out, n)
		}
	}
	return out
}

func displayTag(n *VNode) string {
	if n.Kind == KindComponent {
		return "kiw-component:" + n.ComponentName
	}
	return n.Tag
}
