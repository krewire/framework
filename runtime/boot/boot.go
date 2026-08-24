// Package boot hydrates mount points emitted by web/ssg: it scans for
// data-kiw-mount markers, instantiates the registered component, seeds the
// fiber with the server-rendered DOM so the first render diffs instead of
// re-rendering, and drives later renders through a per-frame scheduler
// (KWF-T4X9P FRK-WASM-040/041). Platform-neutral helpers live here; the
// syscall/js driver is boot_js.go.
package boot

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/krewire/framework/runtime/component"
)

// FlattenProps decodes a data-kiw-props JSON payload into component.Props.
// Strings pass through; numbers and booleans are stringified; any other
// shape is JSON-encoded so typed factories always receive a stable scalar
// map. A null or empty payload yields an empty Props.
func FlattenProps(payload []byte) (component.Props, error) {
	if len(trimSpace(payload)) == 0 {
		return component.Props{}, nil
	}
	var raw map[string]any
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil, fmt.Errorf("boot: decode props: %w", err)
	}
	out := make(component.Props, len(raw))
	for k, v := range raw {
		s, err := scalar(v)
		if err != nil {
			return nil, fmt.Errorf("boot: prop %q: %w", k, err)
		}
		out[k] = s
	}
	return out, nil
}

func scalar(v any) (string, error) {
	switch t := v.(type) {
	case nil:
		return "", nil
	case string:
		return t, nil
	case bool:
		return strconv.FormatBool(t), nil
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10), nil
		}
		return strconv.FormatFloat(t, 'g', -1, 64), nil
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
}

func trimSpace(b []byte) []byte {
	start, end := 0, len(b)
	for start < end && isSpace(b[start]) {
		start++
	}
	for end > start && isSpace(b[end-1]) {
		end--
	}
	return b[start:end]
}

func isSpace(c byte) bool { return c == ' ' || c == '\t' || c == '\n' || c == '\r' }
