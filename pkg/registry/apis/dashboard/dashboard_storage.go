package dashboard

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/rest"

	"github.com/grafana/grafana-app-sdk/logging"
	"github.com/grafana/grafana/pkg/apimachinery/identity"
	"github.com/grafana/grafana/pkg/apimachinery/utils"
	grafanarest "github.com/grafana/grafana/pkg/apiserver/rest"
	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/registry/apis/dashboard/home"
	"github.com/grafana/grafana/pkg/services/accesscontrol"
	"github.com/grafana/grafana/pkg/services/apiserver/endpoints/request"
	"github.com/grafana/grafana/pkg/services/featuremgmt"
	"github.com/grafana/grafana/pkg/services/live"
)

// dashboardStorageWrapper is a wrapper around the grafanarest.Storage so it will:
// 1. support adds dashboard permissions handling
// 2. broadcast changes to grafana live
// when running in single tenant mode
type dashboardStorageWrapper struct {
	grafanarest.Storage

	// Support home dashboards
	homeDashboard home.HomeDashboardGetter
	apiVersion    string

	// Clear the dashboard cache on Delete
	dashboardPermissionsSvc accesscontrol.DashboardPermissionsService

	// Broadcast events
	live live.DashboardActivityChannel

	// Skip the legacy permission deletion when the App Platform path owns permissions
	features featuremgmt.FeatureToggles

	log log.Logger
}

func (d dashboardStorageWrapper) Create(ctx context.Context, obj runtime.Object, createValidation rest.ValidateObjectFunc, options *metav1.CreateOptions) (runtime.Object, error) {
	out, err := d.Storage.Create(ctx, obj, createValidation, options)
	if err == nil {
		d.logAudit(ctx, "created", "", out)
	}
	return out, err
}

func (d dashboardStorageWrapper) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	ns, err := request.NamespaceInfoFrom(ctx, true)
	if err != nil {
		return nil, false, err
	}

	obj, created, err := d.Storage.Update(ctx, name, objInfo, createValidation, updateValidation, forceAllowCreate, options)
	if err == nil && ns.OrgID > 0 && d.live != nil {
		m, err := utils.MetaAccessor(obj)
		if err == nil {
			if err := d.live.DashboardSaved(ns.Value, name, m.GetResourceVersion()); err != nil {
				logging.FromContext(ctx).Info("live dashboard update failed", "err", err)
			}
		}
	}
	if err == nil {
		action := "updated"
		if created {
			action = "created"
		}
		d.logAudit(ctx, action, name, obj)
	}
	return obj, created, err
}

func (d dashboardStorageWrapper) Delete(ctx context.Context, name string, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	ns, err := request.NamespaceInfoFrom(ctx, true)
	if err != nil {
		return nil, false, err
	}
	obj, async, err := d.Storage.Delete(ctx, name, deleteValidation, options)
	if err != nil {
		return obj, async, err
	}
	d.logAudit(ctx, "deleted", name, obj)
	if ns.OrgID > 0 && d.live != nil {
		if err := d.live.DashboardDeleted(ns.Value, name); err != nil {
			logging.FromContext(ctx).Info("live dashboard update failed", "err", err)
		}
	}
	// With the flag on, the App Platform path (the store's afterDelete hook) deletes permissions, so
	// skip the legacy deletion here to avoid a double call. Standalone never registers this wrapper.
	if d.features.IsEnabledGlobally(featuremgmt.FlagKubernetesAuthzResourcePermissionApis) { //nolint:staticcheck
		return obj, async, nil
	}
	if accessErr := d.dashboardPermissionsSvc.DeleteResourcePermissions(ctx, ns.OrgID, name); accessErr != nil {
		return obj, async, accessErr
	}
	return obj, async, nil
}

func (d dashboardStorageWrapper) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	if name == home.DASHBOARD_NAME && d.homeDashboard != nil {
		return d.homeDashboard.Get(d.apiVersion)
	}

	return d.Storage.Get(ctx, name, options)
}

func (d dashboardStorageWrapper) logAudit(ctx context.Context, action string, dashboardUID string, obj runtime.Object) {
	if d.log == nil {
		return
	}

	ns, _ := request.NamespaceInfoFrom(ctx, true)
	uid := dashboardUID
	if uid == "" && obj != nil {
		if m, err := utils.MetaAccessor(obj); err == nil {
			uid = m.GetName()
		}
	}

	var actorUID, actorLogin string
	if requester, err := identity.GetRequester(ctx); err == nil {
		actorUID = requester.GetUID()
		actorLogin = requester.GetLogin()
	}

	d.log.Info("Dashboard audit log",
		"action", action,
		"dashboardUid", uid,
		"orgId", ns.OrgID,
		"namespace", ns.Value,
		"actorUid", actorUID,
		"actorLogin", actorLogin,
	)
}
