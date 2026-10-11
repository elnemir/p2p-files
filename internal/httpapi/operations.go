package httpapi

import (
	"filex.local/filex/internal/auth"
	"filex.local/filex/internal/config"
	"filex.local/filex/internal/reports"
	"net/http"
	"strconv"
)

func (a *App) operationsRoutes(m *http.ServeMux) {
	m.HandleFunc("GET /api/v1/admin/operations/audit-evidence", a.protect("admin:operations", true, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		digest, e := a.Operations.AuditEvidence(r.Context())
		if e != nil {
			return e
		}
		return reply(w, 200, map[string]string{"evidenceHash": digest})
	}))
	m.HandleFunc("GET /api/v1/admin/operations", a.protect("admin:operations", true, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		state, e := a.Operations.State(r.Context())
		if e != nil {
			return e
		}
		jobs, e := a.Operations.Jobs(r.Context())
		if e != nil {
			return e
		}
		incidents, e := a.Operations.Incidents(r.Context())
		if e != nil {
			return e
		}
		checkpoints, e := a.Operations.Checkpoints(r.Context())
		if e != nil {
			return e
		}
		schedules, e := a.Operations.Schedules(r.Context())
		if e != nil {
			return e
		}
		return reply(w, 200, map[string]any{"state": state, "jobs": jobs, "incidents": incidents, "checkpoints": checkpoints, "schedules": schedules})
	}))
	m.HandleFunc("POST /api/v1/admin/operations/maintenance", a.protect("admin:operations", true, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		var v struct {
			Reason string `json:"reason"`
		}
		if e := body(w, r, &v); e != nil {
			return e
		}
		if e := a.Operations.SetMaintenance(r.Context(), p.ID, v.Reason); e != nil {
			return e
		}
		w.WriteHeader(204)
		return nil
	}))
	m.HandleFunc("POST /api/v1/admin/operations/jobs", a.protect("admin:operations", true, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		var v struct {
			Kind     string `json:"kind"`
			Source   string `json:"sourceId"`
			Provider string `json:"providerId"`
			Reason   string `json:"reason"`
		}
		if e := body(w, r, &v); e != nil {
			return e
		}
		id, e := a.Operations.Queue(r.Context(), p.ID, v.Kind, v.Source, v.Provider, v.Reason)
		if e != nil {
			return e
		}
		return reply(w, 202, map[string]string{"id": id})
	}))
	m.HandleFunc("GET /api/v1/admin/operations/reconciliations/{id}", a.protect("admin:operations", true, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		raw, e := a.Operations.Reconciliation(r.Context(), r.PathValue("id"))
		if e != nil {
			return e
		}
		return reply(w, 200, raw)
	}))
	m.HandleFunc("POST /api/v1/admin/operations/reopen", a.protect("admin:operations", true, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		var v struct {
			ID     string `json:"reconciliationId"`
			Hash   string `json:"hash"`
			Reason string `json:"reason"`
			Ack    bool   `json:"acknowledgeCapabilities"`
		}
		if e := body(w, r, &v); e != nil {
			return e
		}
		if e := a.Operations.Reopen(r.Context(), p.ID, v.ID, v.Hash, v.Reason, v.Ack); e != nil {
			return e
		}
		w.WriteHeader(204)
		return nil
	}))
	m.HandleFunc("POST /api/v1/admin/operations/capabilities/revoke", a.protect("admin:operations", true, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		var v struct {
			Reason string `json:"reason"`
		}
		if e := body(w, r, &v); e != nil {
			return e
		}
		if e := a.Operations.RevokeCapabilities(r.Context(), p.ID, v.Reason); e != nil {
			return e
		}
		w.WriteHeader(204)
		return nil
	}))
	m.HandleFunc("POST /api/v1/admin/operations/checkpoints", a.protect("admin:operations", true, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		var v struct {
			Kind   string `json:"kind"`
			Reason string `json:"reason"`
			Hash   string `json:"evidenceHash"`
		}
		if e := body(w, r, &v); e != nil {
			return e
		}
		out, e := a.Operations.Checkpoint(r.Context(), p.ID, v.Kind, v.Reason, v.Hash)
		if e != nil {
			return e
		}
		return reply(w, 201, out)
	}))
	m.HandleFunc("PATCH /api/v1/admin/operations/incidents/{id}", a.protect("admin:operations", true, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		var v struct {
			State  string `json:"state"`
			Reason string `json:"reason"`
		}
		if e := body(w, r, &v); e != nil {
			return e
		}
		if e := a.Operations.UpdateIncident(r.Context(), r.PathValue("id"), v.State, p.ID, v.Reason); e != nil {
			return e
		}
		w.WriteHeader(204)
		return nil
	}))
	m.HandleFunc("GET /api/v1/admin/operations/incidents/{id}/report", a.protect("admin:operations", true, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		revision := 0
		var e error
		if v := r.URL.Query().Get("revision"); v != "" {
			revision, e = strconv.Atoi(v)
			if e != nil || revision < 0 {
				return auth.ErrInvalid
			}
		}
		format := r.URL.Query().Get("format")
		if format == "" {
			format = "json"
		}
		if format != "json" && format != "pdf" {
			return auth.ErrInvalid
		}
		raw, digest, e := a.Operations.IncidentReport(r.Context(), r.PathValue("id"), revision)
		if e != nil {
			return e
		}
		w.Header().Set("X-Content-SHA256", digest)
		if format == "pdf" {
			w.Header().Set("Content-Type", "application/pdf")
			w.Header().Set("Content-Disposition", `attachment; filename="security-report.pdf"`)
			return reports.SecurityPDF(w, raw, digest)
		}
		w.Header().Set("Content-Type", "application/json")
		_, e = w.Write(raw)
		return e
	}))
	m.HandleFunc("PUT /api/v1/admin/operations/schedules", a.protect("admin:operations", true, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		var v struct {
			Kind     string `json:"kind"`
			Interval int    `json:"intervalSeconds"`
			Provider string `json:"providerId"`
		}
		if e := body(w, r, &v); e != nil {
			return e
		}
		if e := a.Operations.SetSchedule(r.Context(), p.ID, v.Kind, v.Interval, v.Provider); e != nil {
			return e
		}
		w.WriteHeader(204)
		return nil
	}))
}
