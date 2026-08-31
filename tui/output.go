package tui

import (
	"encoding/json"
	"io"
)

// RenderJSON marshals v to stable JSON for machine-readable CLI output
// (KWF-NPFSE). Struct fields serialize in declaration order and map keys are
// sorted by encoding/json, so the result is deterministic across runs.
func RenderJSON(v any) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

// WriteJSON writes v as deterministic JSON to w (typically os.Stdout), keeping
// structured results on the data stream, separate from diagnostics on stderr.
func WriteJSON(w io.Writer, v any) error {
	b, err := RenderJSON(v)
	if err != nil {
		return err
	}
	_, err = w.Write(append(b, '\n'))
	return err
}

// Result is a minimal structured command outcome for JSON/machine output.
type Result struct {
	// OK reports whether the command succeeded.
	OK bool `json:"ok"`
	// Data carries the command's payload when OK is true.
	Data any `json:"data,omitempty"`
	// Error carries a human-readable message when OK is false.
	Error string `json:"error,omitempty"`
}

// ResultOK builds a successful Result.
func ResultOK(data any) Result {
	return Result{OK: true, Data: data}
}

// ResultErr builds a failed Result from msg.
func ResultErr(msg string) Result {
	return Result{OK: false, Error: msg}
}
