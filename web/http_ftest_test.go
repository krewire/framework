// Tests for KWF-TEST-M4P9Q
package web

import (
	"net/http"
	"testing"

	ftest "github.com/krewire/framework/test"
)

// Spec: KWF-TEST-M4P9Q KWF-TST-M4P-051 Scope: Package
func TestKWF_TST_M4P_051_Migrated_HTTP_UsingFtest(t *testing.T) {
	ftest.Spec(t, "KWF-TEST-M4P9Q", "KWF-TST-M4P-051")
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("krewire"))
	})
	req := ftest.NewRequest(t, "GET", "/", nil)
	rec := ftest.Record(handler, req)
	ftest.EqualStatus(t, rec, http.StatusOK)
	ftest.Contains(t, rec.Body.String(), "krewire")
	ftest.Equal(t, rec.Body.String(), "krewire")
}
