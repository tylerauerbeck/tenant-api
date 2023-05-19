package pubsub

import (
	"testing"

	natssrv "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.infratographer.com/x/gidx"
	"go.uber.org/zap"
)

var natsSrv *natssrv.Server

func TestMain(m *testing.M) {
	srv, err := StartNatsServer()
	if err != nil {
		panic(err)
	}

	natsSrv = srv

	defer natsSrv.Shutdown()

	m.Run()
}

func TestClient_AddStream(t *testing.T) {
	id, err := gidx.NewID("testing")

	require.NoError(t, err, "expected no error creating new id")

	nc, err := nats.Connect(natsSrv.ClientURL())
	if err != nil {
		// fail open on nats
		t.Log(err)
	}

	js, err := nc.JetStream()
	if err != nil {
		// fail open on nats
		t.Log(err)
	}

	c1 := NewClient(
		WithJetreamContext(js),
		WithLogger(zap.NewNop().Sugar()),
		WithStreamName("nats-test-"+string(id)),
		WithSubjectPrefix("com.infratographer."+string(id)),
	)

	out, err := c1.AddStream()

	assert.NoError(t, err)

	defer func() {
		err := c1.deleteStream()
		assert.NoError(t, err)
	}()

	assert.Equal(t, "nats-test-"+string(id), out.Config.Name)
	assert.Equal(t, nats.FileStorage, out.Config.Storage)
	assert.Equal(t, nats.LimitsPolicy, out.Config.Retention)
	assert.Equal(t, nats.DiscardNew, out.Config.Discard)

	// run AddStream a second time, no error should be returned
	_, err = c1.AddStream()
	assert.NoError(t, err)

	c2 := NewClient(
		WithJetreamContext(js),
		WithLogger(zap.NewNop().Sugar()),
		WithStreamName("nats-test-overlaps"),
		WithSubjectPrefix("com.infratographer."+string(id)),
	)

	// AddStream should error since subjects overlap
	_, err = c2.AddStream()
	assert.Error(t, err)
}
