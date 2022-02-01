package mqttclient_test

import (
	"context"
	"os"

	mc "github.com/SB-IM/charoite/pkg/mqttclient"
	"github.com/rs/zerolog/log"
	"github.com/williamlsh/logging"
)

func ExampleMQTTClient() {
	logging.Debug(true)

	if err := os.Setenv("DEBUG_MQTT_CLIENT", "true"); err != nil {
		log.Fatal().Err(err)
	}
	defer func() {
		if err := os.Unsetenv("DEBUG_MQTT_CLIENT"); err != nil {
			log.Fatal().Err(err)
		}
	}()

	ctx := log.Logger.WithContext(context.Background())
	client := mc.NewClient(ctx, mc.ConfigOptions{})

	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	//Output: Noop
}
