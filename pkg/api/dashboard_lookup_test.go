package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana/pkg/infra/db/dbtest"
	"github.com/grafana/grafana/pkg/services/accesscontrol/actest"
	"github.com/grafana/grafana/pkg/setting"
)

func TestLookupDashboards(t *testing.T) {
	mockSQLStore := dbtest.NewFakeDB()

	loggedInUserScenario(t, "When calling GET with uid on", "/api/dashboards/lookup", "/api/dashboards/lookup", func(sc *scenarioContext) {
		hs := &HTTPServer{
			Cfg:           setting.NewCfg(),
			SQLStore:      mockSQLStore,
			AccessControl: actest.FakeAccessControl{ExpectedEvaluate: true},
		}
		sc.handlerFunc = hs.LookupDashboards
		sc.fakeReqWithParams("GET", sc.url, map[string]string{"uid": "dash"}).exec()

		require.Equal(t, http.StatusOK, sc.resp.Code)

		var body struct {
			Dashboards []lookupDashboardDTO `json:"dashboards"`
		}
		err := json.NewDecoder(sc.resp.Body).Decode(&body)
		require.NoError(t, err)
		assert.Empty(t, body.Dashboards)
	}, mockSQLStore)

	loggedInUserScenario(t, "When calling GET without uid or tag on", "/api/dashboards/lookup", "/api/dashboards/lookup", func(sc *scenarioContext) {
		hs := &HTTPServer{
			Cfg:           setting.NewCfg(),
			SQLStore:      mockSQLStore,
			AccessControl: actest.FakeAccessControl{ExpectedEvaluate: true},
		}
		sc.handlerFunc = hs.LookupDashboards
		sc.fakeReqWithParams("GET", sc.url, map[string]string{}).exec()

		assert.Equal(t, http.StatusBadRequest, sc.resp.Code)
	}, mockSQLStore)
}

func TestBuildLookupDashboardsQuery(t *testing.T) {
	t.Run("scopes to org, excludes folders and deleted dashboards, and applies a limit", func(t *testing.T) {
		query, args := buildLookupDashboardsQuery(2, "abc", "")
		require.Contains(t, query, "is_folder = FALSE")
		require.NotContains(t, query, "is_folder = 0")
		require.Contains(t, query, "deleted IS NULL")
		require.Contains(t, query, "org_id = ?")
		require.Contains(t, query, "LIMIT ?")
		require.Equal(t, []any{int64(2), "abc", lookupDashboardsMaxResults}, args)
	})

	t.Run("tag lookup is still org-scoped and limited", func(t *testing.T) {
		query, args := buildLookupDashboardsQuery(7, "", "shared")
		require.Contains(t, query, "org_id = ?")
		require.Contains(t, query, "dashboard_tag")
		require.Contains(t, query, "LIMIT ?")
		require.Equal(t, []any{int64(7), "shared", lookupDashboardsMaxResults}, args)
	})
}
