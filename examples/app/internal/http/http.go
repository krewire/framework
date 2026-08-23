// Package http contains the application's HTTP layer (FRK-STR-003): handlers
// and routes for the JSON API.
package http

import (
	"net/http"

	rvapp "github.com/krewire/framework/app"
	rvweb "github.com/krewire/framework/web"
	"github.com/krewire/libs/validate"
)

// Handler serves the JSON API endpoints.
type Handler struct{}

// NewHandler builds the API handler.
func NewHandler() *Handler { return &Handler{} }

// Status reports service health.
func (h *Handler) Status(w http.ResponseWriter, _ *http.Request, _ rvweb.Params) {
	type statusResp struct {
		Status  string `json:"status"`
		Version string `json:"version"`
	}
	rvweb.JSON(w, http.StatusOK, statusResp{Status: "ok", Version: "0.6.0"})
}

// Contact validates and stores a contact request.
func (h *Handler) Contact(w http.ResponseWriter, r *http.Request, _ rvweb.Params) {
	type contactReq struct {
		Email string `json:"email" validate:"required,email"`
		Name  string `json:"name" validate:"required,len=3"`
	}
	var req contactReq
	if err := rvweb.ReadJSON(r, &req); err != nil {
		rvweb.Error(w, err)
		return
	}
	if err := validate.Struct(&req); err != nil {
		rvweb.Error(w, &rvweb.HTTPError{Status: http.StatusBadRequest, Code: "invalid", Message: err.Error()})
		return
	}
	rvweb.JSON(w, http.StatusCreated, map[string]string{"email": req.Email})
}

// Provider registers the API handler and its routes (KWF-D1CNT).
type Provider struct{}

// Register binds the handler into the container.
func (p *Provider) Register(c *rvapp.Container) error {
	return rvapp.Singleton[*Handler](c, func() *Handler { return NewHandler() })
}

// Boot mounts the API routes once the web app is assembled.
func (p *Provider) Boot(c *rvapp.Container) error {
	a, err := rvapp.Resolve[*rvweb.App](c)
	if err != nil {
		return err
	}
	h, err := rvapp.Resolve[*Handler](c)
	if err != nil {
		return err
	}
	a.Router().Get("/api/status", h.Status)
	a.Router().Post("/api/contact", h.Contact)
	return nil
}
