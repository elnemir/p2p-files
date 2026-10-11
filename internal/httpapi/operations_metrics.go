package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func (a *App) operationsMetrics(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	state, e := a.Operations.State(ctx)
	if e != nil {
		writeStatus(w, 503, "unavailable")
		return
	}
	var critical, warning, failed, pending int
	var backup float64
	e = a.DB.QueryRow(ctx, `SELECT (SELECT count(*) FROM security_incidents WHERE severity='critical' AND state<>'resolved'),(SELECT count(*) FROM security_incidents WHERE severity='warning' AND state<>'resolved'),(SELECT count(*) FROM ops_jobs WHERE state='failed'),(SELECT count(*) FROM ops_jobs WHERE state IN ('queued','running')),(SELECT COALESCE(extract(epoch FROM max(completed_at)),0) FROM ops_jobs WHERE kind='backup' AND state='completed')`).Scan(&critical, &warning, &failed, &pending, &backup)
	if e != nil {
		writeStatus(w, 503, "unavailable")
		return
	}
	open := 0
	if state.Mode == "open" && state.Generation == a.Gate.Generation {
		open = 1
	}
	var text strings.Builder
	fmt.Fprintf(&text, "# TYPE filex_authenticated_requests_total counter\nfilex_authenticated_requests_total %d\n", a.requests.Load())
	fmt.Fprintf(&text, "# TYPE filex_data_plane_open gauge\nfilex_data_plane_open %d\n", open)
	fmt.Fprintf(&text, "# TYPE filex_security_incidents gauge\nfilex_security_incidents{severity=\"critical\"} %d\nfilex_security_incidents{severity=\"warning\"} %d\n", critical, warning)
	fmt.Fprintf(&text, "# TYPE filex_operations_jobs gauge\nfilex_operations_jobs{state=\"failed\"} %d\nfilex_operations_jobs{state=\"pending\"} %d\n", failed, pending)
	fmt.Fprintf(&text, "# TYPE filex_backup_last_success_timestamp_seconds gauge\nfilex_backup_last_success_timestamp_seconds %.3f\n", backup)
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	_, _ = w.Write([]byte(text.String()))
}
