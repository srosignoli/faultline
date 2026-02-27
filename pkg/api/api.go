package api

import (
	"context"
	"encoding/json"
	"net/http"
)

// SimulatorClient is the interface the API layer uses to manage simulators.
// *k8s.Client satisfies this interface.
type SimulatorClient interface {
	CreateSimulator(ctx context.Context, name, dumpPayload, rulesPayload string) error
	ListSimulators(ctx context.Context) ([]string, error)
	DeleteSimulator(ctx context.Context, name string) error
}

// Handler holds the dependencies for the HTTP handlers.
type Handler struct {
	k8s SimulatorClient
}

// NewHandler creates a Handler with the given SimulatorClient.
func NewHandler(k8s SimulatorClient) *Handler {
	return &Handler{k8s: k8s}
}

// NewRouter wires up all API routes onto a new ServeMux and returns it.
// Uses Go 1.22+ method+path routing syntax.
func NewRouter(h *Handler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/simulators", h.listSimulators)
	mux.HandleFunc("POST /api/simulators", h.createSimulator)
	mux.HandleFunc("DELETE /api/simulators/{name}", h.deleteSimulator)
	return mux
}

// --- request / response types ---

type createRequest struct {
	Name         string `json:"name"`
	DumpPayload  string `json:"dump_payload"`
	RulesPayload string `json:"rules_payload"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// --- helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse{Error: msg})
}

// --- handlers ---

func (h *Handler) listSimulators(w http.ResponseWriter, r *http.Request) {
	names, err := h.k8s.ListSimulators(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, names)
}

func (h *Handler) createSimulator(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.DumpPayload == "" {
		writeError(w, http.StatusBadRequest, "dump_payload is required")
		return
	}
	if req.RulesPayload == "" {
		writeError(w, http.StatusBadRequest, "rules_payload is required")
		return
	}
	if err := h.k8s.CreateSimulator(r.Context(), req.Name, req.DumpPayload, req.RulesPayload); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) deleteSimulator(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if err := h.k8s.DeleteSimulator(r.Context(), name); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
