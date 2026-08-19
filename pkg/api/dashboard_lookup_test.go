package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana/pkg/apimachinery/identity"
	"github.com/grafana/grafana/pkg/infra/db/dbtest"
	"github.com/grafana/grafana/pkg/services/accesscontrol/acimpl"
	"github.com/grafana/grafana/pkg/services/authn"
	"github.com/grafana/grafana/pkg/services/authn/authntest"
	contextmodel "github.com/grafana/grafana/pkg/services/contexthandler/model"
	"github.com/grafana/grafana/pkg/services/dashboards"
	"github.com/grafana/grafana/pkg/services/org"
	"github.com/grafana/grafana/pkg/services/org/orgtest"
	"github.com/grafana/grafana/pkg/services/sqlstore/migrator"
	"github.com/grafana/grafana/pkg/services/user"
	"github.com/grafana/grafana/pkg/setting"
	"github.com/grafana/grafana/pkg/web"
)

func TestLookupDashboards(t *testing.T) {
	mockSQLStore := dbtest.NewFakeDB()

	loggedInUserScenario(t, "When calling GET with uid on", "/api/dashboards/lookup", "/api/dashboards/lookup", func(sc *scenarioContext) {
		hs := &HTTPServer{
			Cfg:      setting.NewCfg(),
			SQLStore: mockSQLStore,
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
			Cfg:      setting.NewCfg(),
			SQLStore: mockSQLStore,
		}
		sc.handlerFunc = hs.LookupDashboards
		sc.fakeReqWithParams("GET", sc.url, map[string]string{}).exec()

		assert.Equal(t, http.StatusBadRequest, sc.resp.Code)
	}, mockSQLStore)

	loggedInUserScenario(t, "When calling GET with orgId the user does not belong to on", "/api/dashboards/lookup", "/api/dashboards/lookup", func(sc *scenarioContext) {
		hs := &HTTPServer{
			Cfg:      setting.NewCfg(),
			SQLStore: mockSQLStore,
			orgService: &orgtest.FakeOrgService{
				ExpectedUserOrgDTO: []*org.UserOrgDTO{{OrgID: testOrgID}},
			},
		}
		sc.handlerFunc = hs.LookupDashboards
		sc.fakeReqWithParams("GET", sc.url, map[string]string{"uid": "dash", "orgId": "99"}).exec()

		require.Equal(t, http.StatusOK, sc.resp.Code)

		var body struct {
			Dashboards []lookupDashboardDTO `json:"dashboards"`
		}
		err := json.NewDecoder(sc.resp.Body).Decode(&body)
		require.NoError(t, err)
		assert.Empty(t, body.Dashboards)
	}, mockSQLStore)
}

func TestBuildLookupDashboardsQuery(t *testing.T) {
	t.Run("uses dialect boolean and excludes deleted rows with a limit", func(t *testing.T) {
		query, args := buildLookupDashboardsQuery(migrator.NewPostgresDialect(), "dash", "prod", []int64{1, 2})

		require.Contains(t, query, "is_folder = ?")
		require.Contains(t, query, "deleted IS NULL")
		require.Contains(t, query, "org_id IN (?,?)")
		require.Contains(t, query, "LIMIT 1000")
		require.Equal(t, false, args[0])
		require.Equal(t, "dash", args[1])
		require.Equal(t, int64(1), args[2])
		require.Equal(t, int64(2), args[3])
		require.Equal(t, "prod", args[4])
	})

	t.Run("uses integer boolean values on sqlite", func(t *testing.T) {
		query, args := buildLookupDashboardsQuery(migrator.NewSQLite3Dialect(), "dash", "", []int64{1})

		require.Contains(t, query, "is_folder = ?")
		require.Contains(t, query, "deleted IS NULL")
		require.Contains(t, query, "LIMIT 1000")
		require.Equal(t, 0, args[0])
	})
}

func TestLookupAllowedOrgIDs(t *testing.T) {
	hs := &HTTPServer{
		orgService: &orgtest.FakeOrgService{
			ExpectedUserOrgDTO: []*org.UserOrgDTO{
				{OrgID: 1},
				{OrgID: 2},
			},
		},
	}

	t.Run("includes every org the user belongs to", func(t *testing.T) {
		c := lookupReqContext(1, 1, nil)
		orgIDs, err := hs.lookupAllowedOrgIDs(context.Background(), c)
		require.NoError(t, err)
		assert.Equal(t, []int64{1, 2}, orgIDs)
	})

	t.Run("pins to a requested org the user belongs to", func(t *testing.T) {
		c := lookupReqContext(1, 1, map[string]string{"orgId": "2"})
		orgIDs, err := hs.lookupAllowedOrgIDs(context.Background(), c)
		require.NoError(t, err)
		assert.Equal(t, []int64{2}, orgIDs)
	})

	t.Run("rejects a requested org the user does not belong to", func(t *testing.T) {
		c := lookupReqContext(1, 1, map[string]string{"orgId": "99"})
		orgIDs, err := hs.lookupAllowedOrgIDs(context.Background(), c)
		require.NoError(t, err)
		assert.Empty(t, orgIDs)
	})
}

func TestCanReadLookupDashboard(t *testing.T) {
	hs := &HTTPServer{
		AccessControl: acimpl.ProvideAccessControlTest(),
	}

	t.Run("allows dashboards the caller can read in the current org", func(t *testing.T) {
		c := lookupReqContext(1, 1, nil)
		c.Permissions = map[int64]map[string][]string{
			1: {dashboards.ActionDashboardsRead: {dashboards.ScopeDashboardsAll}},
		}
		orgUsers := map[int64]identity.Requester{1: c.SignedInUser}

		ok, err := hs.canReadLookupDashboard(context.Background(), c, lookupDashboardDTO{OrgID: 1, UID: "dash"}, orgUsers)
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("denies dashboards without dashboards:read", func(t *testing.T) {
		c := lookupReqContext(1, 1, nil)
		c.Permissions = map[int64]map[string][]string{1: {}}
		orgUsers := map[int64]identity.Requester{1: c.SignedInUser}

		ok, err := hs.canReadLookupDashboard(context.Background(), c, lookupDashboardDTO{OrgID: 1, UID: "dash"}, orgUsers)
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("evaluates access in the destination org", func(t *testing.T) {
		hs := &HTTPServer{
			AccessControl: acimpl.ProvideAccessControlTest(),
			authnService: &authntest.FakeService{
				ExpectedIdentity: &authn.Identity{
					OrgID: 2,
					Permissions: map[int64]map[string][]string{
						2: {dashboards.ActionDashboardsRead: {dashboards.ScopeDashboardsAll}},
					},
				},
			},
		}
		c := lookupReqContext(1, 1, nil)
		c.Permissions = map[int64]map[string][]string{1: {}}
		orgUsers := map[int64]identity.Requester{1: c.SignedInUser}

		ok, err := hs.canReadLookupDashboard(context.Background(), c, lookupDashboardDTO{OrgID: 2, UID: "dash"}, orgUsers)
		require.NoError(t, err)
		assert.True(t, ok)
	})
}

func lookupReqContext(userID, orgID int64, params map[string]string) *contextmodel.ReqContext {
	req := httptest.NewRequest(http.MethodGet, "/api/dashboards/lookup", nil)
	if len(params) > 0 {
		q := req.URL.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}
	ctx := &contextmodel.ReqContext{
		Context: &web.Context{Req: req},
		SignedInUser: &user.SignedInUser{
			UserID: userID,
			OrgID:  orgID,
		},
	}
	return ctx
}
