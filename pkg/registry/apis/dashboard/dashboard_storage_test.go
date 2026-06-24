package dashboard

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"

	"github.com/grafana/grafana/pkg/apimachinery/identity"
	grafanarest "github.com/grafana/grafana/pkg/apiserver/rest"
	"github.com/grafana/grafana/pkg/infra/log/logtest"
	acmock "github.com/grafana/grafana/pkg/services/accesscontrol/mock"
	"github.com/grafana/grafana/pkg/services/featuremgmt"
)

func testAuditContext(t *testing.T) (context.Context, *identity.StaticRequester) {
	t.Helper()
	ctx := apirequest.WithNamespace(context.Background(), "default")
	requester := &identity.StaticRequester{
		UserUID: "user-uid-1",
		Login:   "testuser",
		OrgID:   1,
	}
	return identity.WithRequester(ctx, requester), requester
}

func newAuditTestWrapper(t *testing.T, storage grafanarest.Storage, fake *logtest.Fake) dashboardStorageWrapper {
	t.Helper()
	return dashboardStorageWrapper{
		Storage:                 storage,
		dashboardPermissionsSvc:   &acmock.MockPermissionsService{},
		features:                featuremgmt.WithFeatures(featuremgmt.FlagKubernetesAuthzResourcePermissionApis),
		log:                     fake,
	}
}

func auditCtxValue(ctx []any, key string) any {
	for i := 0; i+1 < len(ctx); i += 2 {
		if ctx[i] == key {
			return ctx[i+1]
		}
	}
	return nil
}

func requireAuditLog(t *testing.T, fake *logtest.Fake, requester *identity.StaticRequester, action, dashboardUID string) {
	t.Helper()
	require.Equal(t, 1, fake.InfoLogs.Calls)
	require.Equal(t, "Dashboard audit log", fake.InfoLogs.Message)
	require.Equal(t, action, auditCtxValue(fake.InfoLogs.Ctx, "action"))
	require.Equal(t, dashboardUID, auditCtxValue(fake.InfoLogs.Ctx, "dashboardUid"))
	require.Equal(t, int64(1), auditCtxValue(fake.InfoLogs.Ctx, "orgId"))
	require.Equal(t, "default", auditCtxValue(fake.InfoLogs.Ctx, "namespace"))
	require.Equal(t, requester.GetUID(), auditCtxValue(fake.InfoLogs.Ctx, "actorUid"))
	require.Equal(t, "testuser", auditCtxValue(fake.InfoLogs.Ctx, "actorLogin"))
}

func TestDashboardStorageWrapperDelete(t *testing.T) {
	// "default" maps to orgID 1, which is required by NamespaceInfoFrom.
	ctx := apirequest.WithNamespace(context.Background(), "default")

	newWrapper := func(features featuremgmt.FeatureToggles, perms *acmock.MockPermissionsService) dashboardStorageWrapper {
		storage := grafanarest.NewMockStorage(t)
		storage.On("Delete", mock.Anything, "dash-uid", mock.Anything, mock.Anything).
			Return(&unstructured.Unstructured{}, false, nil)
		return dashboardStorageWrapper{
			Storage:                 storage,
			dashboardPermissionsSvc: perms,
			features:                features,
		}
	}

	t.Run("flag off deletes legacy permissions", func(t *testing.T) {
		perms := &acmock.MockPermissionsService{}
		perms.On("DeleteResourcePermissions", mock.Anything, int64(1), "dash-uid").Return(nil)
		w := newWrapper(featuremgmt.WithFeatures(), perms)

		_, _, err := w.Delete(ctx, "dash-uid", nil, &metav1.DeleteOptions{})
		require.NoError(t, err)
		perms.AssertCalled(t, "DeleteResourcePermissions", mock.Anything, int64(1), "dash-uid")
	})

	t.Run("flag on skips legacy permission deletion (handled by the afterDelete hook)", func(t *testing.T) {
		perms := &acmock.MockPermissionsService{}
		w := newWrapper(featuremgmt.WithFeatures(featuremgmt.FlagKubernetesAuthzResourcePermissionApis), perms)

		_, _, err := w.Delete(ctx, "dash-uid", nil, &metav1.DeleteOptions{})
		require.NoError(t, err)
		perms.AssertNotCalled(t, "DeleteResourcePermissions", mock.Anything, mock.Anything, mock.Anything)
	})
}

func TestDashboardStorageWrapperAuditLogging(t *testing.T) {
	ctx, requester := testAuditContext(t)

	t.Run("Create logs created on success", func(t *testing.T) {
		fake := &logtest.Fake{}
		storage := grafanarest.NewMockStorage(t)
		obj := &unstructured.Unstructured{}
		obj.SetName("dash-uid")
		storage.On("Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(obj, nil)
		w := newAuditTestWrapper(t, storage, fake)

		out, err := w.Create(ctx, obj, nil, &metav1.CreateOptions{})
		require.NoError(t, err)
		require.Equal(t, obj, out)
		requireAuditLog(t, fake, requester, "created", "dash-uid")
	})

	t.Run("Create does not log on error", func(t *testing.T) {
		fake := &logtest.Fake{}
		storage := grafanarest.NewMockStorage(t)
		storage.On("Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(nil, errors.New("create failed"))
		w := newAuditTestWrapper(t, storage, fake)

		_, err := w.Create(ctx, &unstructured.Unstructured{}, nil, &metav1.CreateOptions{})
		require.Error(t, err)
		require.Equal(t, 0, fake.InfoLogs.Calls)
	})

	t.Run("Update logs updated on success", func(t *testing.T) {
		fake := &logtest.Fake{}
		storage := grafanarest.NewMockStorage(t)
		obj := &unstructured.Unstructured{}
		obj.SetName("dash-uid")
		storage.On("Update", mock.Anything, "dash-uid", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(obj, false, nil)
		w := newAuditTestWrapper(t, storage, fake)

		out, created, err := w.Update(ctx, "dash-uid", nil, nil, nil, false, &metav1.UpdateOptions{})
		require.NoError(t, err)
		require.False(t, created)
		require.Equal(t, obj, out)
		requireAuditLog(t, fake, requester, "updated", "dash-uid")
	})

	t.Run("Update logs created when storage creates", func(t *testing.T) {
		fake := &logtest.Fake{}
		storage := grafanarest.NewMockStorage(t)
		obj := &unstructured.Unstructured{}
		obj.SetName("dash-uid")
		storage.On("Update", mock.Anything, "dash-uid", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(obj, true, nil)
		w := newAuditTestWrapper(t, storage, fake)

		_, created, err := w.Update(ctx, "dash-uid", nil, nil, nil, false, &metav1.UpdateOptions{})
		require.NoError(t, err)
		require.True(t, created)
		requireAuditLog(t, fake, requester, "created", "dash-uid")
	})

	t.Run("Update does not log on error", func(t *testing.T) {
		fake := &logtest.Fake{}
		storage := grafanarest.NewMockStorage(t)
		storage.On("Update", mock.Anything, "dash-uid", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(nil, false, errors.New("update failed"))
		w := newAuditTestWrapper(t, storage, fake)

		_, _, err := w.Update(ctx, "dash-uid", nil, nil, nil, false, &metav1.UpdateOptions{})
		require.Error(t, err)
		require.Equal(t, 0, fake.InfoLogs.Calls)
	})

	t.Run("Delete logs deleted on success", func(t *testing.T) {
		fake := &logtest.Fake{}
		storage := grafanarest.NewMockStorage(t)
		storage.On("Delete", mock.Anything, "dash-uid", mock.Anything, mock.Anything).
			Return(&unstructured.Unstructured{}, false, nil)
		w := newAuditTestWrapper(t, storage, fake)

		_, _, err := w.Delete(ctx, "dash-uid", nil, &metav1.DeleteOptions{})
		require.NoError(t, err)
		requireAuditLog(t, fake, requester, "deleted", "dash-uid")
	})

	t.Run("Delete does not log on error", func(t *testing.T) {
		fake := &logtest.Fake{}
		storage := grafanarest.NewMockStorage(t)
		storage.On("Delete", mock.Anything, "dash-uid", mock.Anything, mock.Anything).
			Return(nil, false, errors.New("delete failed"))
		w := newAuditTestWrapper(t, storage, fake)

		_, _, err := w.Delete(ctx, "dash-uid", nil, &metav1.DeleteOptions{})
		require.Error(t, err)
		require.Equal(t, 0, fake.InfoLogs.Calls)
	})
}