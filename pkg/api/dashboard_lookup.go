package api

import (
	"fmt"
	"net/http"

	"github.com/grafana/grafana/pkg/api/response"
	"github.com/grafana/grafana/pkg/components/simplejson"
	"github.com/grafana/grafana/pkg/infra/db"
	contextmodel "github.com/grafana/grafana/pkg/services/contexthandler/model"
	"github.com/grafana/grafana/pkg/util"
)

// swagger:route GET /dashboards/lookup dashboards lookupDashboards
//
// Lookup dashboards by UID or tag.
//
// Resolves dashboards for share-link and command-palette flows. Unlike GET /dashboards/uid/{uid},
// this lookup can run before the user has switched organization.
//
// Responses:
// 200: lookupDashboardsResponse
// 400: badRequestError
// 401: unauthorisedError
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

	query := "SELECT id, org_id, uid, title, slug, data FROM dashboard WHERE is_folder = 0"
	args := []any{}

	if uid != "" {
		query += " AND uid = ?"
		args = append(args, uid)
	}

	// Optional org pin for clients that already know the destination org.
	if orgID := c.QueryInt64("orgId"); orgID > 0 {
		query += " AND org_id = ?"
		args = append(args, orgID)
	}

	if tag != "" {
		query += " AND id IN (SELECT dashboard_id FROM dashboard_tag WHERE term = ?)"
		args = append(args, tag)
	}

	dashboards := make([]lookupDashboardDTO, 0)
	err := hs.SQLStore.WithDbSession(ctx, func(sess *db.Session) error {
		if err := sess.SQL(query, args...).Find(&dashboards); err != nil {
			return fmt.Errorf("looking up dashboards: %w", err)
		}
		return nil
	})
	if err != nil {
		return response.Error(http.StatusInternalServerError, "Failed to lookup dashboards", err)
	}

	return response.JSON(http.StatusOK, util.DynMap{
		"dashboards": dashboards,
	})
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
	// Organization ID. When omitted, lookup is not limited to the signed-in user's current org.
	// in:query
	// required:false
	OrgID int64 `json:"orgId"`
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
