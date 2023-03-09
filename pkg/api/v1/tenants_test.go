package api

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTenants(t *testing.T) {
	srv, err := newTestServer()
	defer srv.close()

	require.NoError(t, err, "no error expected for new test server")

	t.Run("no tenants", func(t *testing.T) {
		var result *v1TenantSliceResponse

		resp, err := srv.Request(http.MethodGet, "/v1/tenants", nil, nil, &result)
		require.NoError(t, err, "no error expected for tenant list")
		assert.Equal(t, http.StatusOK, resp.StatusCode, "unexpected status code returned")

		require.NotNil(t, result, "expected tenants result")
		assert.Equal(t, apiVersion, result.Version, "unexpected response version")
		assert.Len(t, result.Tenants, 0, "expected no tenants")
	})

	var t1Resp *v1TenantResponse

	t.Run("new tenant", func(t *testing.T) {
		createRequest := strings.NewReader(`{"name": "tenant1"}`)

		resp, err := srv.Request(http.MethodPost, "/v1/tenants", nil, createRequest, &t1Resp)
		require.NoError(t, err, "no error expected for creating tenant")
		assert.Equal(t, http.StatusCreated, resp.StatusCode, "unexpected status code returned")
		assert.NotEmpty(t, t1Resp.Tenant.ID, "expected tenant id")
		assert.Equal(t, "tenant1", t1Resp.Tenant.Name, "unexpected tenant name")
	})

	var t1aResp *v1TenantResponse

	t.Run("new subtenant", func(t *testing.T) {
		createRequest := strings.NewReader(`{"name": "tenant1.a"}`)

		resp, err := srv.Request(http.MethodPost, "/v1/tenants/"+t1Resp.Tenant.ID+"/tenants", nil, createRequest, &t1aResp)
		require.NoError(t, err, "no error expected for creating subtenant")
		assert.Equal(t, http.StatusCreated, resp.StatusCode, "unexpected status code returned")
		assert.NotEmpty(t, t1aResp.Tenant.ID, "expected tenant id")
		assert.Equal(t, "tenant1.a", t1aResp.Tenant.Name, "unexpected tenant name")
		require.NotNil(t, t1aResp.Tenant.ParentTenantID, "expected parent tenant id to be set")
		assert.Equal(t, t1Resp.Tenant.ID, *t1aResp.Tenant.ParentTenantID, "unexpected parent tenant id")
	})

	t.Run("list tenants", func(t *testing.T) {
		var result *v1TenantSliceResponse

		resp, err := srv.Request(http.MethodGet, "/v1/tenants", nil, nil, &result)
		require.NoError(t, err, "no error expected for tenant list")
		assert.Equal(t, http.StatusOK, resp.StatusCode, "unexpected status code returned")

		require.Len(t, result.Tenants, 1, "expected 1 tenant")
		assert.Equal(t, t1Resp.Tenant.ID, result.Tenants[0].ID, "expected tenant1 id")
		assert.Equal(t, t1Resp.Tenant.Name, result.Tenants[0].Name, "expected tenant1 name")
	})

	t.Run("list subtenants", func(t *testing.T) {
		var result *v1TenantSliceResponse

		resp, err := srv.Request(http.MethodGet, "/v1/tenants/"+t1Resp.Tenant.ID+"/tenants", nil, nil, &result)
		require.NoError(t, err, "no error expected for tenant list")
		assert.Equal(t, http.StatusOK, resp.StatusCode, "unexpected status code returned")

		require.Len(t, result.Tenants, 1, "expected 1 tenant")
		assert.Equal(t, t1aResp.Tenant.ID, result.Tenants[0].ID, "expected tenant1.a id")
		assert.Equal(t, t1aResp.Tenant.Name, result.Tenants[0].Name, "expected tenant1.a name")
	})
}
