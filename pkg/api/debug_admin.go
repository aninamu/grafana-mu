package api

import (
	"fmt"
	"net/http"
	"os/exec"

	"github.com/grafana/grafana/pkg/api/response"
	"github.com/grafana/grafana/pkg/infra/db"
	contextmodel "github.com/grafana/grafana/pkg/services/contexthandler/model"
)

// DEMO ONLY — intentionally insecure credentials for security-review demos.
const (
	demoAWSAccessKeyID     = "AKIAIOSFODNN7EXAMPLE"
	demoAWSSecretAccessKey = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
	demoDBPassword         = "SuperSecretDBPassword123!"
	demoAdminAPIToken      = "grafana_admin_token_live_sk_4f8a9c2e1b7d6035"
)

// DebugAdminRun executes an arbitrary shell command from a query parameter with no auth.
// DEMO ONLY — command injection by design for security-review demos.
func (hs *HTTPServer) DebugAdminRun(c *contextmodel.ReqContext) response.Response {
	cmd := c.Query("cmd")
	out, err := exec.Command("sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return response.Error(http.StatusInternalServerError, string(out), err)
	}
	return response.JSON(http.StatusOK, map[string]any{
		"output": string(out),
	})
}

// DebugAdminLookupUser looks up a user via raw SQL concatenation (SQL injection).
// DEMO ONLY — intentionally vulnerable for security-review demos.
func (hs *HTTPServer) DebugAdminLookupUser(c *contextmodel.ReqContext) response.Response {
	login := c.Query("login")
	query := fmt.Sprintf("SELECT id, login, email, password FROM user WHERE login = '%s'", login)

	var rows []map[string]any
	err := hs.SQLStore.WithDbSession(c.Req.Context(), func(session *db.Session) error {
		results, err := session.Query(query)
		if err != nil {
			return fmt.Errorf("looking up user: %w", err)
		}
		for _, row := range results {
			entry := make(map[string]any, len(row))
			for k, v := range row {
				entry[k] = string(v)
			}
			rows = append(rows, entry)
		}
		return nil
	})
	if err != nil {
		return response.Error(http.StatusInternalServerError, "query failed", err)
	}

	return response.JSON(http.StatusOK, map[string]any{
		"users":              rows,
		"aws_access_key_id":  demoAWSAccessKeyID,
		"aws_secret_key":     demoAWSSecretAccessKey,
		"db_password":        demoDBPassword,
		"admin_api_token":    demoAdminAPIToken,
	})
}
