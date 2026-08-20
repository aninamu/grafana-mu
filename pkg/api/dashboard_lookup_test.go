package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana/pkg/infra/db/dbtest"
	"github.com/grafana/grafana/pkg/services/accesscontrol"
	"github.com/grafana/grafana/pkg/services/authn"
	"github.com/grafana/grafana/pkg/services/authn/authntest"
	"github.com/grafana/grafana/pkg/services/dashboards"
	"github.com/grafana/grafana/pkg/services/dashboards/dashboardaccess"
	"github.com/grafana/grafana/pkg/services/org"
	"github.com/grafana/grafana/pkg/services/org/orgtest"
	"github.com/grafana/grafana/pkg/services/search/model"
	"github.com/grafana/grafana/pkg/setting"
)

func TestLookupDashboards(t *testing.T) {
	mockSQLStore := dbtest.NewFakeDB()

	loggedInUserScenario(t, "When calling GET with uid on", "/api/dashboards/lookup", "/api/dashboards/lookup", func(sc *scenarioContext) {
		dashSvc := dashboards.NewFakeDashboardService(t)
		dashSvc.On("SearchDashboards", mock.Anything, mock.MatchedBy(func(q *dashboards.FindPersistedDashboardsQuery) bool {
			return q.OrgId == testOrgID &&
				q.Limit == lookupDashboardsMaxResults &&
				q.Type == model.TypeDashboard &&
				q.Permission == dashboardaccess.PERMISSION_VIEW &&
				len(q.DashboardUIDs) == 1 && q.DashboardUIDs[0] == "dash" &&
				len(q.Tags) == 0
		})).Return(model.HitList{}, nil).Once()

		hs := &HTTPServer{
			Cfg:              setting.NewCfg(),
			SQLStore:         mockSQLStore,
			DashboardService: dashSvc,
			orgService:       &orgtest.FakeOrgService{ExpectedUserOrgDTO: []*org.UserOrgDTO{{OrgID: testOrgID}}},
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

	loggedInUserScenario(t, "When calling GET with tag on", "/api/dashboards/lookup", "/api/dashboards/lookup", func(sc *scenarioContext) {
		dashSvc := dashboards.NewFakeDashboardService(t)
		dashSvc.On("SearchDashboards", mock.Anything, mock.MatchedBy(func(q *dashboards.FindPersistedDashboardsQuery) bool {
			return q.OrgId == testOrgID &&
				q.Limit == lookupDashboardsMaxResults &&
				q.Type == model.TypeDashboard &&
				q.Permission == dashboardaccess.PERMISSION_VIEW &&
				len(q.Tags) == 1 && q.Tags[0] == "shared" &&
				len(q.DashboardUIDs) == 0
		})).Return(model.HitList{}, nil).Once()

		hs := &HTTPServer{
			Cfg:              setting.NewCfg(),
			SQLStore:         mockSQLStore,
			DashboardService: dashSvc,
			orgService:       &orgtest.FakeOrgService{ExpectedUserOrgDTO: []*org.UserOrgDTO{{OrgID: testOrgID}}},
		}
		sc.handlerFunc = hs.LookupDashboards
		sc.fakeReqWithParams("GET", sc.url, map[string]string{"tag": "shared"}).exec()

		require.Equal(t, http.StatusOK, sc.resp.Code)
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

	loggedInUserScenario(t, "When calling GET with uid and orgId the caller belongs to on", "/api/dashboards/lookup", "/api/dashboards/lookup", func(sc *scenarioContext) {
		const destOrgID int64 = 2
		dashSvc := dashboards.NewFakeDashboardService(t)
		dashSvc.On("SearchDashboards", mock.Anything, mock.MatchedBy(func(q *dashboards.FindPersistedDashboardsQuery) bool {
			return q.OrgId == destOrgID &&
				len(q.DashboardUIDs) == 1 && q.DashboardUIDs[0] == "dash"
		})).Return(model.HitList{}, nil).Once()

		hs := &HTTPServer{
			Cfg:              setting.NewCfg(),
			SQLStore:         mockSQLStore,
			DashboardService: dashSvc,
			orgService: &orgtest.FakeOrgService{ExpectedUserOrgDTO: []*org.UserOrgDTO{
				{OrgID: testOrgID},
				{OrgID: destOrgID},
			}},
			authnService: &authntest.FakeService{ExpectedIdentity: &authn.Identity{}},
		}
		sc.handlerFunc = hs.LookupDashboards
		sc.fakeReqWithParams("GET", sc.url, map[string]string{"uid": "dash", "orgId": "2"}).exec()

		require.Equal(t, http.StatusOK, sc.resp.Code)
		dashSvc.AssertNumberOfCalls(t, "SearchDashboards", 1)
	}, mockSQLStore)

	loggedInUserScenario(t, "When calling GET with uid and orgId the caller does not belong to on", "/api/dashboards/lookup", "/api/dashboards/lookup", func(sc *scenarioContext) {
		dashSvc := dashboards.NewFakeDashboardService(t)

		hs := &HTTPServer{
			Cfg:              setting.NewCfg(),
			SQLStore:         mockSQLStore,
			DashboardService: dashSvc,
			orgService:       &orgtest.FakeOrgService{ExpectedUserOrgDTO: []*org.UserOrgDTO{{OrgID: testOrgID}}},
			authnService:     &authntest.FakeService{ExpectedIdentity: &authn.Identity{}},
		}
		sc.handlerFunc = hs.LookupDashboards
		sc.fakeReqWithParams("GET", sc.url, map[string]string{"uid": "dash", "orgId": "2"}).exec()

		assert.Equal(t, http.StatusForbidden, sc.resp.Code)
		dashSvc.AssertNotCalled(t, "SearchDashboards")
	}, mockSQLStore)

	loggedInUserScenario(t, "When calling GET with uid and orgId and identity resolve fails on", "/api/dashboards/lookup", "/api/dashboards/lookup", func(sc *scenarioContext) {
		const destOrgID int64 = 2
		dashSvc := dashboards.NewFakeDashboardService(t)

		hs := &HTTPServer{
			Cfg:              setting.NewCfg(),
			SQLStore:         mockSQLStore,
			DashboardService: dashSvc,
			orgService: &orgtest.FakeOrgService{ExpectedUserOrgDTO: []*org.UserOrgDTO{
				{OrgID: testOrgID},
				{OrgID: destOrgID},
			}},
			authnService: &authntest.FakeService{
				ExpectedIdentity: &authn.Identity{},
				ExpectedErr:      errors.New("resolve failed"),
			},
		}
		sc.handlerFunc = hs.LookupDashboards
		sc.fakeReqWithParams("GET", sc.url, map[string]string{"uid": "dash", "orgId": "2"}).exec()

		assert.Equal(t, http.StatusForbidden, sc.resp.Code)
		dashSvc.AssertNotCalled(t, "SearchDashboards")
	}, mockSQLStore)

	loggedInUserScenario(t, "When calling GET with uid and orgId and identity is not an org member on", "/api/dashboards/lookup", "/api/dashboards/lookup", func(sc *scenarioContext) {
		const destOrgID int64 = 2
		dashSvc := dashboards.NewFakeDashboardService(t)

		hs := &HTTPServer{
			Cfg:              setting.NewCfg(),
			SQLStore:         mockSQLStore,
			DashboardService: dashSvc,
			orgService: &orgtest.FakeOrgService{ExpectedUserOrgDTO: []*org.UserOrgDTO{
				{OrgID: testOrgID},
				{OrgID: destOrgID},
			}},
			authnService: &authntest.FakeService{
				ExpectedIdentities: []*authn.Identity{{OrgID: accesscontrol.NoOrgID}},
				ExpectedErrs:       []error{nil},
			},
		}
		sc.handlerFunc = hs.LookupDashboards
		sc.fakeReqWithParams("GET", sc.url, map[string]string{"uid": "dash", "orgId": "2"}).exec()

		assert.Equal(t, http.StatusForbidden, sc.resp.Code)
		dashSvc.AssertNotCalled(t, "SearchDashboards")
	}, mockSQLStore)

	loggedInUserScenario(t, "When calling GET with uid across user orgs on", "/api/dashboards/lookup", "/api/dashboards/lookup", func(sc *scenarioContext) {
		const otherOrgID int64 = 2
		dashSvc := dashboards.NewFakeDashboardService(t)
		dashSvc.On("SearchDashboards", mock.Anything, mock.MatchedBy(func(q *dashboards.FindPersistedDashboardsQuery) bool {
			return q.OrgId == testOrgID
		})).Return(model.HitList{}, nil).Once()
		dashSvc.On("SearchDashboards", mock.Anything, mock.MatchedBy(func(q *dashboards.FindPersistedDashboardsQuery) bool {
			return q.OrgId == otherOrgID && len(q.DashboardUIDs) == 1 && q.DashboardUIDs[0] == "dash"
		})).Return(model.HitList{}, nil).Once()

		hs := &HTTPServer{
			Cfg:              setting.NewCfg(),
			SQLStore:         mockSQLStore,
			DashboardService: dashSvc,
			orgService: &orgtest.FakeOrgService{ExpectedUserOrgDTO: []*org.UserOrgDTO{
				{OrgID: testOrgID},
				{OrgID: otherOrgID},
			}},
			authnService: &authntest.FakeService{ExpectedIdentity: &authn.Identity{}},
		}
		sc.handlerFunc = hs.LookupDashboards
		sc.fakeReqWithParams("GET", sc.url, map[string]string{"uid": "dash"}).exec()

		require.Equal(t, http.StatusOK, sc.resp.Code)
		dashSvc.AssertNumberOfCalls(t, "SearchDashboards", 2)
	}, mockSQLStore)
}

func TestBuildLookupDashboardsByUIDsQuery(t *testing.T) {
	t.Run("scopes to org, excludes folders and deleted dashboards, and filters by authorized UIDs", func(t *testing.T) {
		query, args := buildLookupDashboardsByUIDsQuery(2, []string{"abc", "def"})
		require.Contains(t, query, "is_folder = FALSE")
		require.NotContains(t, query, "is_folder = 0")
		require.Contains(t, query, "deleted IS NULL")
		require.Contains(t, query, "org_id = ?")
		require.Contains(t, query, "uid IN (?,?)")
		require.Contains(t, query, "ORDER BY id")
		require.NotContains(t, query, "LIMIT")
		require.Equal(t, []any{int64(2), "abc", "def"}, args)
	})
}
