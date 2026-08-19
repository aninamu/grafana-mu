package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/grafana/grafana/pkg/api/response"
	"github.com/grafana/grafana/pkg/components/simplejson"
	"github.com/grafana/grafana/pkg/infra/db"
	contextmodel "github.com/grafana/grafana/pkg/services/contexthandler/model"
	"github.com/grafana/grafana/pkg/services/dashboards"
	"github.com/grafana/grafana/pkg/services/dashboards/dashboardaccess"
	"github.com/grafana/grafana/pkg/services/search/model"
	"github.com/grafana/grafana/pkg/util"
)

const lookupDashboardsMaxResults = 1000

// swagger:route GET /dashboards/lookup dashboards lookupDashboards
//
// Lookup dashboards by UID or tag.
//
// Resolves dashboards for share-link and command-palette flows.
// Results are limited to non-deleted dashboards in the signed-in user's current
// organization that the caller can read.
//
// Responses:
// 200: lookupDashboardsResponse
// 400: badRequestError
// 401: unauthorisedError
// 403: forbiddenError
// 500: internalServerError
func (hs *HTTPServer) LookupDashboards(c *contextmodel.ReqContext) response.Response {
	ctx, span := tracer.Start(c.Req.Context(), "api.LookupDashboards")
	defer span.End()
	c.Req = c.Req.WithContext(ctx)

	uid := c.Query("uid")
	tag := c.Query("tag")
	if uid == "" && tag == "" {
		return response.Error(http.StatusBadRequest, "uid or tag query parameter is required", nil)
	}

	searchQuery := dashboards.FindPersistedDashboardsQuery{
		OrgId:        c.GetOrgID(),
		SignedInUser: c.SignedInUser,
		Limit:        lookupDashboardsMaxResults,
		Type:         model.TypeDashboard,
		Permission:   dashboardaccess.PERMISSION_VIEW,
	}
	if uid != "" {
		searchQuery.DashboardUIDs = []string{uid}
	}
	if tag != "" {
		searchQuery.Tags = []string{tag}
	}

	hits, err := hs.DashboardService.SearchDashboards(ctx, &searchQuery)
	if err != nil {
		return response.Error(http.StatusInternalServerError, "Failed to lookup dashboards", err)
	}

	visible := make([]lookupDashboardDTO, 0, len(hits))
	if len(hits) == 0 {
		return response.JSON(http.StatusOK, util.DynMap{
			"dashboards": visible,
		})
	}

	uids := make([]string, len(hits))
	for i, hit := range hits {
		uids[i] = hit.UID
	}

	query, args := buildLookupDashboardsByUIDsQuery(c.GetOrgID(), uids)
	err = hs.SQLStore.WithDbSession(ctx, func(sess *db.Session) error {
		if err := sess.SQL(query, args...).Find(&visible); err != nil {
			return fmt.Errorf("looking up dashboards: %w", err)
		}
		return nil
	})
	if err != nil {
		return response.Error(http.StatusInternalServerError, "Failed to lookup dashboards", err)
	}

	return response.JSON(http.StatusOK, util.DynMap{
		"dashboards": visible,
	})
}

func buildLookupDashboardsByUIDsQuery(orgID int64, uids []string) (string, []any) {
	query := "SELECT id, org_id, uid, title, slug, data FROM dashboard WHERE is_folder = FALSE AND deleted IS NULL AND org_id = ?"
	args := make([]any, 0, 1+len(uids))
	args = append(args, orgID)
	query += " AND uid IN (?" + strings.Repeat(",?", len(uids)-1) + ")"
	for _, uid := range uids {
		args = append(args, uid)
	}
	query += " ORDER BY id"
	return query, args
}

type lookupDashboardDTO struct {
	ID    int64            `json:"id" xorm:"id"`
	OrgID int64            `json:"orgId" xorm:"org_id"`
	UID   string           `json:"uid" xorm:"uid"`
	Title string           `json:"title" xorm:"title"`
	Slug  string           `json:"slug" xorm:"slug"`
	Data  *simplejson.Json `json:"data" xorm:"data"`
}

// swagger:parameters lookupDashboards
type LookupDashboardsParams struct {
	// Dashboard UID to resolve.
	// in:query
	// required:false
	UID string `json:"uid"`
	// Dashboard tag to match.
	// in:query
	// required:false
	Tag string `json:"tag"`
}

// swagger:response lookupDashboardsResponse
type LookupDashboardsResponse struct {
	// in: body
	Body LookupDashboardsResponseBody `json:"body"`
}

// swagger:model
type LookupDashboardsResponseBody struct {
	Dashboards []lookupDashboardDTO `json:"dashboards"`
}
