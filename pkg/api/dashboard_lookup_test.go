package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana/pkg/components/simplejson"
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
	}, nil)

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
			DashboardService: dashSvc,
			orgService:       &orgtest.FakeOrgService{ExpectedUserOrgDTO: []*org.UserOrgDTO{{OrgID: testOrgID}}},
		}
		sc.handlerFunc = hs.LookupDashboards
		sc.fakeReqWithParams("GET", sc.url, map[string]string{"tag": "shared"}).exec()

		require.Equal(t, http.StatusOK, sc.resp.Code)
	}, nil)

	loggedInUserScenario(t, "When calling GET without uid or tag on", "/api/dashboards/lookup", "/api/dashboards/lookup", func(sc *scenarioContext) {
		hs := &HTTPServer{
			Cfg: setting.NewCfg(),
		}
		sc.handlerFunc = hs.LookupDashboards
		sc.fakeReqWithParams("GET", sc.url, map[string]string{}).exec()

		assert.Equal(t, http.StatusBadRequest, sc.resp.Code)
	}, nil)

	loggedInUserScenario(t, "When calling GET with uid and orgId on", "/api/dashboards/lookup", "/api/dashboards/lookup", func(sc *scenarioContext) {
		const destOrgID int64 = 2
		dashSvc := dashboards.NewFakeDashboardService(t)
		dashSvc.On("SearchDashboards", mock.Anything, mock.MatchedBy(func(q *dashboards.FindPersistedDashboardsQuery) bool {
			return q.OrgId == destOrgID &&
				len(q.DashboardUIDs) == 1 && q.DashboardUIDs[0] == "dash"
		})).Return(model.HitList{{UID: "dash", OrgID: destOrgID}}, nil).Once()
		dashSvc.On("GetDashboards", mock.Anything, mock.MatchedBy(func(q *dashboards.GetDashboardsQuery) bool {
			return q.OrgID == destOrgID && len(q.DashboardUIDs) == 1 && q.DashboardUIDs[0] == "dash"
		})).Return([]*dashboards.Dashboard{{
			ID:    10,
			OrgID: destOrgID,
			UID:   "dash",
			Title: "Shared dash",
			Slug:  "shared-dash",
			Data:  simplejson.NewFromAny(map[string]any{"title": "Shared dash"}),
		}}, nil).Once()

		hs := &HTTPServer{
			Cfg:              setting.NewCfg(),
			DashboardService: dashSvc,
			authnService:     &authntest.FakeService{ExpectedIdentity: &authn.Identity{}},
		}
		sc.handlerFunc = hs.LookupDashboards
		sc.fakeReqWithParams("GET", sc.url, map[string]string{"uid": "dash", "orgId": "2"}).exec()

		require.Equal(t, http.StatusOK, sc.resp.Code)

		var body struct {
			Dashboards []lookupDashboardDTO `json:"dashboards"`
		}
		err := json.NewDecoder(sc.resp.Body).Decode(&body)
		require.NoError(t, err)
		require.Len(t, body.Dashboards, 1)
		assert.Equal(t, destOrgID, body.Dashboards[0].OrgID)
		assert.Equal(t, "dash", body.Dashboards[0].UID)
		assert.Equal(t, "Shared dash", body.Dashboards[0].Title)
		require.NotNil(t, body.Dashboards[0].Data)
	}, nil)

	loggedInUserScenario(t, "When calling GET with uid across user orgs on", "/api/dashboards/lookup", "/api/dashboards/lookup", func(sc *scenarioContext) {
		const otherOrgID int64 = 2
		dashSvc := dashboards.NewFakeDashboardService(t)
		dashSvc.On("SearchDashboards", mock.Anything, mock.MatchedBy(func(q *dashboards.FindPersistedDashboardsQuery) bool {
			return q.OrgId == testOrgID
		})).Return(model.HitList{}, nil).Once()
		dashSvc.On("SearchDashboards", mock.Anything, mock.MatchedBy(func(q *dashboards.FindPersistedDashboardsQuery) bool {
			return q.OrgId == otherOrgID && len(q.DashboardUIDs) == 1 && q.DashboardUIDs[0] == "dash"
		})).Return(model.HitList{{UID: "dash", OrgID: otherOrgID}}, nil).Once()
		dashSvc.On("GetDashboards", mock.Anything, mock.MatchedBy(func(q *dashboards.GetDashboardsQuery) bool {
			return q.OrgID == otherOrgID && len(q.DashboardUIDs) == 1 && q.DashboardUIDs[0] == "dash"
		})).Return([]*dashboards.Dashboard{{
			ID:    11,
			OrgID: otherOrgID,
			UID:   "dash",
			Title: "Other org dash",
			Slug:  "other-org-dash",
			Data:  simplejson.NewFromAny(map[string]any{"title": "Other org dash"}),
		}}, nil).Once()

		hs := &HTTPServer{
			Cfg:              setting.NewCfg(),
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

		var body struct {
			Dashboards []lookupDashboardDTO `json:"dashboards"`
		}
		err := json.NewDecoder(sc.resp.Body).Decode(&body)
		require.NoError(t, err)
		require.Len(t, body.Dashboards, 1)
		assert.Equal(t, otherOrgID, body.Dashboards[0].OrgID)
		assert.Equal(t, "Other org dash", body.Dashboards[0].Title)
	}, nil)
}
