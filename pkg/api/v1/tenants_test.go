package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	nats "github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.infratographer.com/tenant-api/internal/pubsub"
	"go.infratographer.com/x/echojwtx"
	"go.infratographer.com/x/pubsubx"
)

const (
	natsMsgSubTimeout   = 2 * time.Second
	tenantSubjectCreate = "com.infratographer.events.tenants.create.global"
	tenantSubjectUpdate = "com.infratographer.events.tenants.update.global"
	tenantSubjectDelete = "com.infratographer.events.tenants.delete.global"
	tenantBaseURN       = "urn:infratographer:tenants:"
)

func TestTenantsWithoutAuth(t *testing.T) {
	srv, err := newTestServer(t, nil)
	defer srv.close()

	require.NoError(t, err, "no error expected for new test server")

	t.Run("no tenants", func(t *testing.T) {
		var result *v1TenantSliceResponse

		resp, err := srv.Request(http.MethodGet, "/v1/tenants", nil, nil, &result)
		resp.Body.Close() //nolint:errcheck // Not needed
		require.NoError(t, err, "no error expected for tenant list")
		assert.Equal(t, http.StatusOK, resp.StatusCode, "unexpected status code returned")

		require.NotNil(t, result, "expected tenants result")
		assert.Equal(t, apiVersion, result.Version, "unexpected response version")
		assert.Len(t, result.Tenants, 0, "expected no tenants")
	})

	subscriber := newPubSubClient(t, srv.logger, srv.nats.ClientURL())
	msgChan := make(chan *nats.Msg, 10)

	// create a new nats subscription on the server created above
	subscription, err := subscriber.ChanSubscribe(
		context.TODO(),
		"com.infratographer.events.tenants.>",
		msgChan,
		"tenant-api-test",
	)

	require.NoError(t, err)

	defer func() {
		if err := subscription.Unsubscribe(); err != nil {
			t.Error(err)
		}
	}()

	var t1Resp *v1TenantResponse

	t.Run("new tenant", func(t *testing.T) {
		createRequest := strings.NewReader(`{"name": "tenant1"}`)

		resp, err := srv.Request(http.MethodPost, "/v1/tenants", nil, createRequest, &t1Resp)
		resp.Body.Close() //nolint:errcheck // Not needed
		require.NoError(t, err, "no error expected for creating tenant")
		assert.Equal(t, http.StatusCreated, resp.StatusCode, "unexpected status code returned")
		assert.NotEmpty(t, t1Resp.Tenant.ID, "expected tenant id")
		assert.Equal(t, "tenant1", t1Resp.Tenant.Name, "unexpected tenant name")

		select {
		case msg := <-msgChan:
			pMsg := &pubsubx.Message{}
			err = json.Unmarshal(msg.Data, pMsg)
			require.NoError(t, err)

			assert.Equal(t, tenantSubjectCreate, msg.Subject, "expected nats subject to be tenant create subject")
			assert.Equal(t, "", pMsg.ActorURN, "expected no actor for unauthenticated client")
			assert.Equal(t, pubsub.CreateEventType, pMsg.EventType, "expected event type to be create")
			assert.Equal(t, tenantBaseURN+t1Resp.Tenant.ID, pMsg.SubjectURN, "expected subject urn to be returned tenant urn")
		case <-time.After(natsMsgSubTimeout):
			t.Error("failed to receive nats message")
		}
	})

	var t1aResp *v1TenantResponse

	t.Run("new subtenant", func(t *testing.T) {
		createRequest := strings.NewReader(`{"name": "tenant1.a"}`)

		resp, err := srv.Request(http.MethodPost, "/v1/tenants/"+t1Resp.Tenant.ID+"/tenants", nil, createRequest, &t1aResp)
		resp.Body.Close() //nolint:errcheck // Not needed
		require.NoError(t, err, "no error expected for creating subtenant")
		assert.Equal(t, http.StatusCreated, resp.StatusCode, "unexpected status code returned")
		assert.NotEmpty(t, t1aResp.Tenant.ID, "expected tenant id")
		assert.Equal(t, "tenant1.a", t1aResp.Tenant.Name, "unexpected tenant name")
		require.NotNil(t, t1aResp.Tenant.ParentTenantID, "expected parent tenant id to be set")
		assert.Equal(t, t1Resp.Tenant.ID, *t1aResp.Tenant.ParentTenantID, "unexpected parent tenant id")

		select {
		case msg := <-msgChan:
			pMsg := &pubsubx.Message{}
			err = json.Unmarshal(msg.Data, pMsg)
			assert.NoError(t, err)

			assert.Equal(t, tenantSubjectCreate, msg.Subject, "expected nats subject to be tenant create subject")
			assert.Equal(t, "", pMsg.ActorURN, "expected no actor for unauthenticated client")
			assert.Equal(t, pubsub.CreateEventType, pMsg.EventType, "expected event type to be create")
			assert.Equal(t, tenantBaseURN+t1aResp.Tenant.ID, pMsg.SubjectURN, "expected subject urn to be returned tenant urn")
			require.NotEmpty(t, pMsg.AdditionalSubjectURNs, "expected additional subject urns")
			assert.Contains(t, pMsg.AdditionalSubjectURNs, tenantBaseURN+t1Resp.Tenant.ID, "expected parent urn in additional subject urns")
		case <-time.After(natsMsgSubTimeout):
			t.Error("failed to receive nats message")
		}
	})

	t.Run("list tenants", func(t *testing.T) {
		var result *v1TenantSliceResponse

		resp, err := srv.Request(http.MethodGet, "/v1/tenants", nil, nil, &result)
		resp.Body.Close() //nolint:errcheck // Not needed
		require.NoError(t, err, "no error expected for tenant list")
		assert.Equal(t, http.StatusOK, resp.StatusCode, "unexpected status code returned")

		require.Len(t, result.Tenants, 1, "expected 1 tenant")
		assert.Equal(t, t1Resp.Tenant.ID, result.Tenants[0].ID, "expected tenant1 id")
		assert.Equal(t, t1Resp.Tenant.Name, result.Tenants[0].Name, "expected tenant1 name")
	})

	t.Run("list subtenants", func(t *testing.T) {
		var result *v1TenantSliceResponse

		resp, err := srv.Request(http.MethodGet, "/v1/tenants/"+t1Resp.Tenant.ID+"/tenants", nil, nil, &result)
		resp.Body.Close() //nolint:errcheck // Not needed
		require.NoError(t, err, "no error expected for tenant list")
		assert.Equal(t, http.StatusOK, resp.StatusCode, "unexpected status code returned")

		require.Len(t, result.Tenants, 1, "expected 1 tenant")
		assert.Equal(t, t1aResp.Tenant.ID, result.Tenants[0].ID, "expected tenant1.a id")
		assert.Equal(t, t1aResp.Tenant.Name, result.Tenants[0].Name, "expected tenant1.a name")
	})
}

func TestTenantsWithAuth(t *testing.T) {
	oauthClient, issuer, close := echojwtx.TestOAuthClient("urn:test:user", "")
	defer close()

	srv, err := newTestServer(t, &testServerConfig{
		client: oauthClient,
		auth: &echojwtx.AuthConfig{
			Issuer: issuer,
		},
	})
	defer srv.close()

	require.NoError(t, err, "no error expected for new test server")

	t.Run("no tenants", func(t *testing.T) {
		resp, err := srv.RequestWithClient(http.DefaultClient, http.MethodGet, "/v1/tenants", nil, nil, nil)
		resp.Body.Close() //nolint:errcheck // Not needed
		require.NoError(t, err, "no error expected for tenant list")
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "unexpected status code returned")

		var result *v1TenantSliceResponse

		resp, err = srv.Request(http.MethodGet, "/v1/tenants", nil, nil, &result)
		resp.Body.Close() //nolint:errcheck // Not needed
		require.NoError(t, err, "no error expected for tenant list")
		assert.Equal(t, http.StatusOK, resp.StatusCode, "unexpected status code returned")

		require.NotNil(t, result, "expected tenants result")
		assert.Equal(t, apiVersion, result.Version, "unexpected response version")
		assert.Len(t, result.Tenants, 0, "expected no tenants")
	})

	subscriber := newPubSubClient(t, srv.logger, srv.nats.ClientURL())
	msgChan := make(chan *nats.Msg, 10)

	// create a new nats subscription on the server created above
	subscription, err := subscriber.ChanSubscribe(
		context.TODO(),
		"com.infratographer.events.tenants.>",
		msgChan,
		"tenant-api-test",
	)

	require.NoError(t, err)

	defer func() {
		if err := subscription.Unsubscribe(); err != nil {
			t.Error(err)
		}
	}()

	var t1Resp *v1TenantResponse

	t.Run("new tenant", func(t *testing.T) {
		createRequest := strings.NewReader(`{"name": "tenant1"}`)

		resp, err := srv.RequestWithClient(http.DefaultClient, http.MethodPost, "/v1/tenants", nil, createRequest, nil)
		resp.Body.Close() //nolint:errcheck // Not needed
		require.NoError(t, err, "no error expected for tenant list")
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "unexpected status code returned")

		_, err = createRequest.Seek(0, io.SeekStart)
		assert.NoError(t, err, "no error expected for seek")

		resp, err = srv.Request(http.MethodPost, "/v1/tenants", nil, createRequest, &t1Resp)
		resp.Body.Close() //nolint:errcheck // Not needed
		require.NoError(t, err, "no error expected for creating tenant")
		assert.Equal(t, http.StatusCreated, resp.StatusCode, "unexpected status code returned")
		assert.NotEmpty(t, t1Resp.Tenant.ID, "expected tenant id")
		assert.Equal(t, "tenant1", t1Resp.Tenant.Name, "unexpected tenant name")

		select {
		case msg := <-msgChan:
			pMsg := &pubsubx.Message{}
			err = json.Unmarshal(msg.Data, pMsg)
			require.NoError(t, err)

			assert.Equal(t, tenantSubjectCreate, msg.Subject, "expected nats subject to be tenant create subject")
			assert.Equal(t, "urn:test:user", pMsg.ActorURN, "expected auth subject for actor urn")
			assert.Equal(t, pubsub.CreateEventType, pMsg.EventType, "expected event type to be create")
			assert.Equal(t, tenantBaseURN+t1Resp.Tenant.ID, pMsg.SubjectURN, "expected subject urn to be returned tenant urn")
		case <-time.After(natsMsgSubTimeout):
			t.Error("failed to receive nats message")
		}
	})

	var t1aResp *v1TenantResponse

	t.Run("new subtenant", func(t *testing.T) {
		createRequest := strings.NewReader(`{"name": "tenant1.a"}`)

		resp, err := srv.RequestWithClient(http.DefaultClient, http.MethodPost, "/v1/tenants/"+t1Resp.Tenant.ID+"/tenants", nil, createRequest, nil)
		resp.Body.Close() //nolint:errcheck // Not needed
		require.NoError(t, err, "no error expected for creating subtenant")
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "unexpected status code returned")

		_, err = createRequest.Seek(0, io.SeekStart)
		assert.NoError(t, err, "no error expected for seek")

		resp, err = srv.Request(http.MethodPost, "/v1/tenants/"+t1Resp.Tenant.ID+"/tenants", nil, createRequest, &t1aResp)
		resp.Body.Close() //nolint:errcheck // Not needed
		require.NoError(t, err, "no error expected for creating subtenant")
		assert.Equal(t, http.StatusCreated, resp.StatusCode, "unexpected status code returned")
		assert.NotEmpty(t, t1aResp.Tenant.ID, "expected tenant id")
		assert.Equal(t, "tenant1.a", t1aResp.Tenant.Name, "unexpected tenant name")
		require.NotNil(t, t1aResp.Tenant.ParentTenantID, "expected parent tenant id to be set")
		assert.Equal(t, t1Resp.Tenant.ID, *t1aResp.Tenant.ParentTenantID, "unexpected parent tenant id")

		select {
		case msg := <-msgChan:
			pMsg := &pubsubx.Message{}
			err = json.Unmarshal(msg.Data, pMsg)
			assert.NoError(t, err)

			assert.Equal(t, tenantSubjectCreate, msg.Subject, "expected nats subject to be tenant create subject")
			assert.Equal(t, "urn:test:user", pMsg.ActorURN, "expected auth subject for actor urn")
			assert.Equal(t, pubsub.CreateEventType, pMsg.EventType, "expected event type to be create")
			assert.Equal(t, tenantBaseURN+t1aResp.Tenant.ID, pMsg.SubjectURN, "expected subject urn to be returned tenant urn")
			require.NotEmpty(t, pMsg.AdditionalSubjectURNs, "expected additional subject urns")
			assert.Contains(t, pMsg.AdditionalSubjectURNs, tenantBaseURN+t1Resp.Tenant.ID, "expected parent urn in additional subject urns")
		case <-time.After(natsMsgSubTimeout):
			t.Error("failed to receive nats message")
		}
	})

	t.Run("list tenants", func(t *testing.T) {
		var result *v1TenantSliceResponse

		resp, err := srv.Request(http.MethodGet, "/v1/tenants", nil, nil, &result)
		resp.Body.Close() //nolint:errcheck // Not needed
		require.NoError(t, err, "no error expected for tenant list")
		assert.Equal(t, http.StatusOK, resp.StatusCode, "unexpected status code returned")

		require.Len(t, result.Tenants, 1, "expected 1 tenant")
		assert.Equal(t, t1Resp.Tenant.ID, result.Tenants[0].ID, "expected tenant1 id")
		assert.Equal(t, t1Resp.Tenant.Name, result.Tenants[0].Name, "expected tenant1 name")
	})

	t.Run("list subtenants", func(t *testing.T) {
		var result *v1TenantSliceResponse

		resp, err := srv.Request(http.MethodGet, "/v1/tenants/"+t1Resp.Tenant.ID+"/tenants", nil, nil, &result)
		resp.Body.Close() //nolint:errcheck // Not needed
		require.NoError(t, err, "no error expected for tenant list")
		assert.Equal(t, http.StatusOK, resp.StatusCode, "unexpected status code returned")

		require.Len(t, result.Tenants, 1, "expected 1 tenant")
		assert.Equal(t, t1aResp.Tenant.ID, result.Tenants[0].ID, "expected tenant1.a id")
		assert.Equal(t, t1aResp.Tenant.Name, result.Tenants[0].Name, "expected tenant1.a name")
	})

	t.Run("get tenant", func(t *testing.T) {
		var result *v1TenantResponse

		resp, err := srv.Request(http.MethodGet, "/v1/tenants/"+t1Resp.Tenant.ID, nil, nil, &result)
		resp.Body.Close() //nolint:errcheck // Not needed
		require.NoError(t, err, "no error expected for tenant list")
		assert.Equal(t, http.StatusOK, resp.StatusCode, "unexpected status code returned")

		require.NotEmpty(t, result.Tenant, "expected tenant")
		assert.Equal(t, t1Resp.Tenant.ID, result.Tenant.ID, "expected tenant1 id")
		assert.Equal(t, t1Resp.Tenant.Name, result.Tenant.Name, "expected tenant1 name")
	})

	t.Run("get subtenant", func(t *testing.T) {
		var result *v1TenantResponse

		resp, err := srv.Request(http.MethodGet, "/v1/tenants/"+t1aResp.Tenant.ID, nil, nil, &result)
		resp.Body.Close() //nolint:errcheck // Not needed
		require.NoError(t, err, "no error expected for tenant list")
		assert.Equal(t, http.StatusOK, resp.StatusCode, "unexpected status code returned")

		require.NotEmpty(t, result.Tenant, "expected tenant")
		assert.Equal(t, t1aResp.Tenant.ID, result.Tenant.ID, "expected subtenant id")
		assert.Equal(t, t1aResp.Tenant.Name, result.Tenant.Name, "expected subtenant name")
	})

	t.Run("update tenant", func(t *testing.T) {
		updateRequest := strings.NewReader(`{"name": "tenant1.a-updated"}`)

		resp, err := srv.RequestWithClient(http.DefaultClient, http.MethodPatch, "/v1/tenants/"+t1Resp.Tenant.ID, nil, updateRequest, nil)
		resp.Body.Close() //nolint:errcheck // Not needed
		require.NoError(t, err, "no error expected for updating subtenant")
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "unexpected status code returned")

		_, err = updateRequest.Seek(0, io.SeekStart)
		assert.NoError(t, err, "no error expected for seek")

		resp, err = srv.Request(http.MethodPatch, "/v1/tenants/"+t1Resp.Tenant.ID, nil, updateRequest, &t1aResp)
		resp.Body.Close() //nolint:errcheck // Not needed
		require.NoError(t, err, "no error expected for updating subtenant")
		assert.Equal(t, http.StatusOK, resp.StatusCode, "unexpected status code returned")
		assert.NotEmpty(t, t1aResp.Tenant.ID, "expected tenant id")
		assert.Equal(t, "tenant1.a-updated", t1aResp.Tenant.Name, "unexpected tenant name")
		require.NotNil(t, t1aResp.Tenant.ParentTenantID, "expected parent tenant id to be set")
		assert.Equal(t, t1Resp.Tenant.ID, *t1aResp.Tenant.ParentTenantID, "unexpected parent tenant id")

		select {
		case msg := <-msgChan:
			pMsg := &pubsubx.Message{}
			err = json.Unmarshal(msg.Data, pMsg)
			assert.NoError(t, err)

			assert.Equal(t, tenantSubjectUpdate, msg.Subject, "expected nats subject to be tenant update subject")
			assert.Equal(t, "urn:test:user", pMsg.ActorURN, "expected auth subject for actor urn")
			assert.Equal(t, pubsub.UpdateEventType, pMsg.EventType, "expected event type to be update")
			assert.Equal(t, tenantBaseURN+t1aResp.Tenant.ID, pMsg.SubjectURN, "expected subject urn to be returned tenant urn")
			require.Empty(t, pMsg.AdditionalSubjectURNs, "unexpected additional subject urns")
		case <-time.After(natsMsgSubTimeout):
			t.Error("failed to receive nats message")
		}
	})

	t.Run("delete tenant", func(t *testing.T) {
		resp, err := srv.RequestWithClient(http.DefaultClient, http.MethodDelete, "/v1/tenants/"+t1Resp.Tenant.ID, nil, nil, nil)
		resp.Body.Close() //nolint:errcheck // Not needed
		require.NoError(t, err, "no error expected for updating subtenant")
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "unexpected status code returned")

		resp, err = srv.Request(http.MethodDelete, "/v1/tenants/"+t1Resp.Tenant.ID, nil, nil, nil)
		resp.Body.Close() //nolint:errcheck // Not needed
		require.NoError(t, err, "no error expected for updating subtenant")
		assert.Equal(t, http.StatusOK, resp.StatusCode, "unexpected status code returned")

		select {
		case msg := <-msgChan:
			pMsg := &pubsubx.Message{}
			err = json.Unmarshal(msg.Data, pMsg)
			assert.NoError(t, err)

			assert.Equal(t, tenantSubjectDelete, msg.Subject, "expected nats subject to be tenant delete subject")
			assert.Equal(t, "urn:test:user", pMsg.ActorURN, "expected auth subject for actor urn")
			assert.Equal(t, pubsub.DeleteEventType, pMsg.EventType, "expected event type to be delete")
			assert.Equal(t, tenantBaseURN+t1aResp.Tenant.ID, pMsg.SubjectURN, "expected subject urn to be returned tenant urn")
			require.Empty(t, pMsg.AdditionalSubjectURNs, "unexpected additional subject urns")
		case <-time.After(natsMsgSubTimeout):
			t.Error("failed to receive nats message")
		}
	})

	t.Run("get deleted tenant", func(t *testing.T) {
		var result *v1TenantResponse

		resp, err := srv.Request(http.MethodGet, "/v1/tenants/"+t1aResp.Tenant.ID, nil, nil, &result)
		resp.Body.Close() //nolint:errcheck // Not needed
		require.NoError(t, err, "no error expected for tenant list")
		assert.Equal(t, http.StatusNotFound, resp.StatusCode, "unexpected status code returned")
	})
}
