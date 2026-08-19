package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/grafana/grafana/pkg/api/response"
	"github.com/grafana/grafana/pkg/apimachinery/identity"
	"github.com/grafana/grafana/pkg/components/simplejson"
	"github.com/grafana/grafana/pkg/infra/db"
	"github.com/grafana/grafana/pkg/services/accesscontrol"
	contextmodel "github.com/grafana/grafana/pkg/services/contexthandler/model"
	"github.com/grafana/grafana/pkg/services/dashboards"
	"github.com/grafana/grafana/pkg/services/org"
	"github.com/grafana/grafana/pkg/services/sqlstore/migrator"
	"github.com/grafana/grafana/pkg/util"
)

const lookupDashboardsLimit int64 = 1000

// swagger:route GET /dashboards/lookup dashboards lookupDashboards
//
// Lookup dashboards by UID or tag.
//
// Resolves dashboards for share-link and command-palette flows. Unlike GET /dashboards/uid/{uid},
// this lookup can run before the user has switched organization. Results are limited to
// organizations the caller belongs to and dashboards they can read.
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

	orgIDs, err := hs.lookupAllowedOrgIDs(ctx, c)
	if err != nil {
		return response.Error(http.StatusInternalServerError, "Failed to lookup dashboards", err)
	}
	if len(orgIDs) == 0 {
		return response.JSON(http.StatusOK, util.DynMap{
			"dashboards": []lookupDashboardDTO{},
		})
	}

	found := make([]lookupDashboardDTO, 0)
	err = hs.SQLStore.WithDbSession(ctx, func(sess *db.Session) error {
		query, args := buildLookupDashboardsQuery(hs.SQLStore.GetDialect(), uid, tag, orgIDs)
		if err := sess.SQL(query, args...).Find(&found); err != nil {
			return fmt.Errorf("looking up dashboards: %w", err)
		}
		return nil
	})
	if err != nil {
		return response.Error(http.StatusInternalServerError, "Failed to lookup dashboards", err)
	}

	visible := make([]lookupDashboardDTO, 0, len(found))
	orgUsers := map[int64]identity.Requester{c.GetOrgID(): c.SignedInUser}
	for _, dash := range found {
		ok, err := hs.canReadLookupDashboard(ctx, c, dash, orgUsers)
		if err != nil {
			return response.Error(http.StatusInternalServerError, "Failed to lookup dashboards", err)
		}
		if ok {
			visible = append(visible, dash)
		}
	}

	return response.JSON(http.StatusOK, util.DynMap{
		"dashboards": visible,
	})
}

func (hs *HTTPServer) lookupAllowedOrgIDs(ctx context.Context, c *contextmodel.ReqContext) ([]int64, error) {
	requestedOrgID := c.QueryInt64("orgId")
	orgIDs := []int64{c.GetOrgID()}
	seen := map[int64]struct{}{c.GetOrgID(): {}}

	if hs.orgService != nil && c.UserID > 0 {
		orgs, err := hs.orgService.GetUserOrgList(ctx, &org.GetUserOrgListQuery{UserID: c.UserID})
		if err != nil {
			return nil, fmt.Errorf("listing user organizations: %w", err)
		}
		for _, userOrg := range orgs {
			if _, ok := seen[userOrg.OrgID]; ok {
				continue
			}
			seen[userOrg.OrgID] = struct{}{}
			orgIDs = append(orgIDs, userOrg.OrgID)
		}
	}

	if requestedOrgID > 0 {
		if _, ok := seen[requestedOrgID]; !ok {
			return nil, nil
		}
		return []int64{requestedOrgID}, nil
	}

	return orgIDs, nil
}

func buildLookupDashboardsQuery(dialect migrator.Dialect, uid, tag string, orgIDs []int64) (string, []any) {
	query := "SELECT id, org_id, uid, title, slug, data FROM dashboard WHERE is_folder = ? AND deleted IS NULL"
	args := []any{dialect.BooleanValue(false)}

	if uid != "" {
		query += " AND uid = ?"
		args = append(args, uid)
	}

	query += " AND org_id IN (?" + strings.Repeat(",?", len(orgIDs)-1) + ")"
	for _, orgID := range orgIDs {
		args = append(args, orgID)
	}

	if tag != "" {
		query += " AND id IN (SELECT dashboard_id FROM dashboard_tag WHERE term = ?)"
		args = append(args, tag)
	}

	query += dialect.Limit(lookupDashboardsLimit)
	return query, args
}

func (hs *HTTPServer) canReadLookupDashboard(ctx context.Context, c *contextmodel.ReqContext, dash lookupDashboardDTO, orgUsers map[int64]identity.Requester) (bool, error) {
	if hs.AccessControl == nil {
		return false, nil
	}

	requester, ok := orgUsers[dash.OrgID]
	if !ok {
		if hs.authnService == nil || dash.OrgID == c.GetOrgID() {
			return false, nil
		}
		orgUser, err := hs.authnService.ResolveIdentity(ctx, dash.OrgID, c.GetID())
		if err != nil {
			return false, nil
		}
		if orgUser.GetOrgID() == accesscontrol.NoOrgID {
			return false, nil
		}
		requester = orgUser
		orgUsers[dash.OrgID] = orgUser
	}

	ok, err := hs.AccessControl.Evaluate(ctx, requester, accesscontrol.EvalPermission(
		dashboards.ActionDashboardsRead,
		dashboards.ScopeDashboardsProvider.GetResourceScopeUID(dash.UID),
	))
	if err != nil {
		return false, fmt.Errorf("evaluating dashboard access: %w", err)
	}
	return ok, nil
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
	// Organization ID. When omitted, lookup is limited to organizations the signed-in user belongs to.
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
