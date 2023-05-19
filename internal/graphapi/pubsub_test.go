package graphapi_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.infratographer.com/x/gidx"
	"go.infratographer.com/x/pubsubx"

	"go.infratographer.com/tenant-api/internal/testclient"
)

func TestTenantPubsub(t *testing.T) {
	ctx := context.Background()

	name := gofakeit.DomainName()
	description := gofakeit.Phrase()

	// graphC := graphTestClient(EntClient)
	graphC := graphTestClient(entClientWithPubsubHooks())

	psC := NatsTestClient

	sub, err := psC.PullSubscribe(ctx, "com.infratographer.changes.>", "")
	require.NoError(t, err)

	// create a root tenant and ensure fields are set
	rootResp, err := graphC.TenantCreate(ctx, testclient.CreateTenantInput{
		Name:        name,
		Description: &description,
	})
	require.NoError(t, err)

	rootTenant := rootResp.TenantCreate.Tenant
	msg := getChangeMessage(t, sub)
	assert.Equal(t, "testing-roundtrip-actor", msg.ActorID.String())
	assert.Equal(t, "create", msg.EventType)
	assert.Equal(t, "tenant-api-test", msg.Source)
	assert.Equal(t, rootTenant.ID, msg.SubjectID)
	assert.Empty(t, msg.AdditionalSubjectIDs)
	// expect created_at, updated_at, name, and description changeset
	assert.Len(t, msg.FieldChanges, 4)

	var createdAtVisited, updatedAtVisited, nameVisited, descriptionVisited bool

	for _, change := range msg.FieldChanges {
		assert.Empty(t, change.PreviousValue)

		switch change.Field {
		case "created_at":
			createdAtVisited = true
			ts, err := time.Parse(time.RFC3339, change.CurrentValue)
			assert.NoError(t, err)
			assert.WithinDuration(t, time.Now(), ts, 2*time.Second)
		case "updated_at":
			updatedAtVisited = true
			ts, err := time.Parse(time.RFC3339, change.CurrentValue)
			assert.NoError(t, err)
			assert.WithinDuration(t, time.Now(), ts, 2*time.Second)
		case "name":
			nameVisited = true

			assert.EqualValues(t, name, change.CurrentValue)
		case "description":
			descriptionVisited = true

			assert.EqualValues(t, description, change.CurrentValue)
		default:
			assert.Fail(t, "unexpected field in changeset %s")
			t.Fail()
		}
	}

	assert.True(t, createdAtVisited)
	assert.True(t, updatedAtVisited)
	assert.True(t, nameVisited)
	assert.True(t, descriptionVisited)

	// Add a child tenant with no description
	childResp, err := graphC.TenantCreate(ctx, testclient.CreateTenantInput{
		Name:     "child",
		ParentID: &rootTenant.ID,
	})

	require.NoError(t, err)
	require.NotNil(t, childResp)

	childTnt := childResp.TenantCreate.Tenant

	msg = getChangeMessage(t, sub)
	assert.Equal(t, "testing-roundtrip-actor", msg.ActorID.String())
	assert.Equal(t, "create", msg.EventType)
	assert.Equal(t, "tenant-api-test", msg.Source)
	assert.Equal(t, childTnt.ID, msg.SubjectID)
	assert.EqualValues(t, []gidx.PrefixedID{rootTenant.ID}, msg.AdditionalSubjectIDs)
	// expect created_at, updated_at, name, and description changeset
	assert.Len(t, msg.FieldChanges, 4)

	createdAtVisited = false
	updatedAtVisited = false
	nameVisited = false

	var parentIDVisited bool

	for _, change := range msg.FieldChanges {
		assert.Empty(t, change.PreviousValue)

		switch change.Field {
		case "created_at":
			createdAtVisited = true
			ts, err := time.Parse(time.RFC3339, change.CurrentValue)
			assert.NoError(t, err)
			assert.WithinDuration(t, time.Now(), ts, 2*time.Second)
		case "updated_at":
			updatedAtVisited = true
			ts, err := time.Parse(time.RFC3339, change.CurrentValue)
			assert.NoError(t, err)
			assert.WithinDuration(t, time.Now(), ts, 2*time.Second)
		case "name":
			nameVisited = true

			assert.EqualValues(t, "child", change.CurrentValue)
		case "parent_tenant_id":
			parentIDVisited = true

			assert.EqualValues(t, rootTenant.ID.String(), change.CurrentValue)
		default:
			assert.Fail(t, fmt.Sprintf("unexpected field in changeset %s", change.Field))
			t.Fail()
		}
	}

	assert.True(t, createdAtVisited)
	assert.True(t, updatedAtVisited)
	assert.True(t, nameVisited)
	assert.True(t, parentIDVisited)

	// Update the tenant
	newName := gofakeit.DomainName()
	updatedTenantResp, err := graphC.TenantUpdate(ctx, childTnt.ID, testclient.UpdateTenantInput{Name: &newName})

	require.NoError(t, err)
	require.NotNil(t, updatedTenantResp)

	msg = getChangeMessage(t, sub)
	assert.Equal(t, "testing-roundtrip-actor", msg.ActorID.String())
	assert.Equal(t, "update", msg.EventType)
	assert.Equal(t, "tenant-api-test", msg.Source)
	assert.Equal(t, childTnt.ID, msg.SubjectID)
	assert.EqualValues(t, []gidx.PrefixedID{rootTenant.ID}, msg.AdditionalSubjectIDs)
	// expect updated_at, and name changeset
	assert.Len(t, msg.FieldChanges, 2)

	updatedAtVisited = false
	nameVisited = false

	for _, change := range msg.FieldChanges {
		assert.NotEmpty(t, change.PreviousValue)

		switch change.Field {
		case "updated_at":
			updatedAtVisited = true
			ts, err := time.Parse(time.RFC3339, change.CurrentValue)
			assert.NoError(t, err)
			assert.WithinDuration(t, time.Now(), ts, 2*time.Second)
		case "name":
			nameVisited = true

			assert.EqualValues(t, "child", change.PreviousValue)
			assert.EqualValues(t, newName, change.CurrentValue)
		default:
			assert.Fail(t, "unexpected field in changeset %s")
			t.Fail()
		}
	}

	assert.True(t, updatedAtVisited)
	assert.True(t, nameVisited)

	// delete the child tenant
	_, err = graphC.TenantDelete(ctx, childTnt.ID)
	require.NoError(t, err)

	msg = getChangeMessage(t, sub)
	assert.Equal(t, "testing-roundtrip-actor", msg.ActorID.String())
	assert.Equal(t, "delete", msg.EventType)
	assert.Equal(t, "tenant-api-test", msg.Source)
	assert.Equal(t, childTnt.ID, msg.SubjectID)
	assert.EqualValues(t, []gidx.PrefixedID{rootTenant.ID}, msg.AdditionalSubjectIDs)
	// expect created_at, updated_at, name, and description changeset
	assert.Len(t, msg.FieldChanges, 0)

	// delete the root tenant
	_, err = graphC.TenantDelete(ctx, rootTenant.ID)
	require.NoError(t, err)

	msg = getChangeMessage(t, sub)
	assert.Equal(t, "testing-roundtrip-actor", msg.ActorID.String())
	assert.Equal(t, "delete", msg.EventType)
	assert.Equal(t, "tenant-api-test", msg.Source)
	assert.Equal(t, rootTenant.ID, msg.SubjectID)
	assert.Empty(t, msg.AdditionalSubjectIDs)
	// expect created_at, updated_at, name, and description changeset
	assert.Len(t, msg.FieldChanges, 0)
}

func getChangeMessage(t *testing.T, sub *nats.Subscription) (msg pubsubx.ChangeMessage) {
	msgs, err := sub.Fetch(1)
	require.NoError(t, err)
	require.Len(t, msgs, 1)

	err = json.Unmarshal(msgs[0].Data, &msg)
	require.NoError(t, err)

	return msg
}
