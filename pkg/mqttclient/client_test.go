package mqttclient_test

import (
	"context"
	"testing"

	mc "github.com/SB-IM/charoite/pkg/mqttclient"
	"github.com/rs/zerolog/log"
)

func TestMQTTClientCtx(t *testing.T) {
	ctx := log.Logger.WithContext(context.Background())
	client := mc.NewClient(ctx, mc.ConfigOptions{})
	newCtx := mc.WithContext(ctx, client)
	oldClient := mc.FromContext(newCtx)
	if oldClient == nil {
		t.Fatalf("old client should not be nil")
	}
}
