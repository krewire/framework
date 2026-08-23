package test

import "testing"

func Spec(t *testing.T, specID, reqID string) {
	t.Helper()
	t.Logf("Spec: %s %s", specID, reqID)
}
