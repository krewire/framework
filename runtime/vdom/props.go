package vdom

import "strings"

// BoolAttrs are attributes whose empty value means "present" (rendered bare,
// e.g. <input disabled>) and whose absence means "false". Missing and "" are
// therefore equivalent for these keys during diffing.
var BoolAttrs = map[string]bool{
	"disabled":  true,
	"checked":   true,
	"readonly":  true,
	"required":  true,
	"hidden":    true,
	"autofocus": true,
	"multiple":  true,
	"selected":  true,
	"open":      true,
}

// NormalizeProps applies the package normalization rules to a copy of props
// and returns it. It is the single source of truth shared by Diff and
// RenderHTML.
func NormalizeProps(props map[string]string) map[string]string {
	if len(props) == 0 {
		return nil
	}
	out := make(map[string]string, len(props))
	for k, v := range props {
		switch {
		case k == "class":
			if c := collapseClass(v); c != "" {
				out[k] = c
			}
		case BoolAttrs[k]:
			out[k] = ""
		case v != "":
			out[k] = v
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// propsEqual reports whether two already-normalized prop maps are equivalent.
func propsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if bv, ok := b[k]; !ok || bv != v {
			return false
		}
	}
	return true
}

func collapseClass(v string) string {
	tokens := strings.Fields(v)
	return strings.Join(tokens, " ")
}
