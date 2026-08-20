package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/grafana/grafana/pkg/api/response"
	"github.com/grafana/grafana/pkg/apimachinery/identity"
	"github.com/grafana/grafana/pkg/components/simplejson"
	contextmodel "github.com/grafana/grafana/pkg/services/contexthandler/model"
	"github.com/grafana/grafana/pkg/services/dashboards"
	"github.com/grafana/grafana/pkg/services/dashboards/dashboardaccess"
	"github.com/grafana/grafana/pkg/services/org"
	"github.com/grafana/grafana/pkg/services/search/model"
	"github.com/grafana/grafana/pkg/util"
)

const lookupDashboardsMaxResults = 1000

// swagger:route GET /dashboards/lookup dashboards lookupDashboards
//
// Lookup dashboards by UID or tag.
//
// Resolves dashboards for share-link and command-palette flows. Unlike GET /dashboards/uid/{uid},
// this lookup can run before the user has switched organization. Results are limited to
// non-deleted dashboards in organizations the caller belongs to and can read. An optional
// orgId query parameter pins lookup to one organization.
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

	orgIDs, err := hs.lookupOrgIDs(ctx, c)
	if err != nil {
		return response.Error(http.StatusInternalServerError, "Failed to lookup dashboards", err)
	}

	visible := make([]lookupDashboardDTO, 0)
	for _, orgID := range orgIDs {
		remaining := lookupDashboardsMaxResults - int64(len(visible))
		if remaining <= 0 {
			break
		}

		searchCtx, requester, err := hs.requesterForOrg(ctx, c, orgID)
		if err != nil {
			continue
		}

		searchQuery := dashboards.FindPersistedDashboardsQuery{
			OrgId:        orgID,
			SignedInUser: requester,
			Limit:        remaining,
			Type:         model.TypeDashboard,
			Permission:   dashboardaccess.PERMISSION_VIEW,
		}
		if uid != "" {
			searchQuery.DashboardUIDs = []string{uid}
		}
		if tag != "" {
			searchQuery.Tags = []string{tag}
		}

		hits, err := hs.DashboardService.SearchDashboards(searchCtx, &searchQuery)
		if err != nil {
			return response.Error(http.StatusInternalServerError, "Failed to lookup dashboards", err)
		}
		if len(hits) == 0 {
			continue
		}

		uids := make([]string, len(hits))
		for i, hit := range hits {
			uids[i] = hit.UID
		}

		dashes, err := hs.DashboardService.GetDashboards(searchCtx, &dashboards.GetDashboardsQuery{
			DashboardUIDs: uids,
			OrgID:         orgID,
		})
		if err != nil {
			return response.Error(http.StatusInternalServerError, "Failed to lookup dashboards", err)
		}

		for _, d := range dashes {
			visible = append(visible, lookupDashboardDTO{
				ID:    d.ID,
				OrgID: d.OrgID,
				UID:   d.UID,
				Title: d.Title,
				Slug:  d.Slug,
				Data:  d.Data,
			})
		}
	}

	return response.JSON(http.StatusOK, util.DynMap{
		"dashboards": visible,
	})
}

func (hs *HTTPServer) lookupOrgIDs(ctx context.Context, c *contextmodel.ReqContext) ([]int64, error) {
	if orgID := c.QueryInt64("orgId"); orgID > 0 {
		return []int64{orgID}, nil
	}

	userID, err := c.GetInternalID()
	if err != nil || userID <= 0 {
		return []int64{c.GetOrgID()}, nil
	}

	orgs, err := hs.orgService.GetUserOrgList(ctx, &org.GetUserOrgListQuery{UserID: userID})
	if err != nil {
		return nil, fmt.Errorf("listing user organizations: %w", err)
	}

	orgIDs := make([]int64, 0, len(orgs))
	for _, o := range orgs {
		if o != nil && o.OrgID > 0 {
			orgIDs = append(orgIDs, o.OrgID)
		}
	}
	if len(orgIDs) == 0 {
		return []int64{c.GetOrgID()}, nil
	}
	return orgIDs, nil
}

func (hs *HTTPServer) requesterForOrg(ctx context.Context, c *contextmodel.ReqContext, orgID int64) (context.Context, identity.Requester, error) {
	if orgID == c.GetOrgID() {
		return ctx, c.SignedInUser, nil
	}

	ident, err := hs.authnService.ResolveIdentity(ctx, orgID, c.GetID())
	if err != nil {
		return ctx, nil, fmt.Errorf("resolving identity for org %d: %w", orgID, err)
	}
	return identity.WithRequester(ctx, ident), ident, nil
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
	// Organization ID. When omitted, lookup searches organizations the signed-in user belongs to.
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
