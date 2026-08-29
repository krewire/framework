// Tests for KWF-T4X9P
package schedule

import "testing"

// Spec: KWF-T4X9P FRK-WASM-012 Scope: Unit
func TestFRK_WASM_012_AddBatchesPerFrameAndDedupes(t *testing.T) {
	var fired []string
	var frames int
	var pending func()

	sch := New[string](func(item string) { fired = append(fired, item) })
	sch.Frame(func(fn func()) { frames++; pending = fn })

	sch.Add("a")
	sch.Add("a") // duplicate within the same frame collapses
	sch.Add("b")

	if frames != 1 {
		t.Fatalf("frame requested %d times, want 1", frames)
	}
	if pending == nil || len(fired) != 0 || sch.Len() != 2 {
		t.Fatalf("items must wait for the frame: fired=%v len=%d", fired, sch.Len())
	}

	pending() // simulated animation frame
	if len(fired) != 2 || fired[0] != "a" || fired[1] != "b" {
		t.Fatalf("batch delivery = %v, want [a b]", fired)
	}
	if sch.Len() != 0 {
		t.Fatalf("queue not drained: %d", sch.Len())
	}

	fired = nil
	sch.Add("b") // re-add after flush schedules a fresh frame
	if frames != 2 {
		t.Fatalf("second batch did not request a frame (frames=%d)", frames)
	}
	pending()
	if len(fired) != 1 || fired[0] != "b" {
		t.Fatalf("post-flush delivery = %v", fired)
	}
}

// Spec: KWF-T4X9P FRK-WASM-012 Scope: Unit
func TestFRK_WASM_012_FlushDrainsWithoutFrame(t *testing.T) {
	var fired []int
	sch := New[int](func(item int) { fired = append(fired, item) })
	if DefaultFrame() != nil {
		t.Fatal("native DefaultFrame must be nil so tests stay deterministic")
	}

	sch.Add(7)
	sch.Add(7)
	sch.Add(9)
	if sch.Len() != 2 {
		t.Fatalf("dedupe failed, len=%d", sch.Len())
	}
	sch.Flush()
	if len(fired) != 2 || fired[0] != 7 || fired[1] != 9 {
		t.Fatalf("flush delivery = %v, want [7 9]", fired)
	}
	sch.Flush() // idempotent when empty
	if len(fired) != 2 {
		t.Fatalf("second flush re-delivered: %v", fired)
	}
}

// Spec: KWF-T4X9P FRK-WASM-012 Scope: Unit
func TestFRK_WASM_012_ReentrantAddDuringFlushIsQueued(t *testing.T) {
	var fired []string
	var sch *Scheduler[string]
	sch = New[string](func(item string) {
		fired = append(fired, item)
		if item == "a" {
			sch.Add("c") // enqueued while draining must not be lost
		}
	})
	sch.Add("a")
	sch.Add("b")
	sch.Flush()
	if len(fired) != 3 || fired[2] != "c" {
		t.Fatalf("reentrant add lost: %v", fired)
	}
}
