// mqtt_client is an highly customized mqtt client build upon github.com/eclipse/paho.mqtt.golang.
// There are several different client options for subscriber clients and publisher clients.
// Mainly, `opts.OnConnect`, `CleanSession` and `client.AddRoute` for subscribers.
package mqttclient

import (
	"context"
	stdlog "log"
	"os"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

func init() {
	// Reading from environment.
	if env := os.Getenv("DEBUG_MQTT_CLIENT"); strings.ToLower(env) == "true" {
		// MQTT internal logging.
		mqtt.ERROR = stdlog.New(os.Stdout, "[ERROR] ", 0)
		mqtt.CRITICAL = stdlog.New(os.Stdout, "[CRITICAL] ", 0)
		mqtt.WARN = stdlog.New(os.Stdout, "[WARN]  ", 0)
		mqtt.DEBUG = stdlog.New(os.Stdout, "[DEBUG] ", 0)
	}
}

type contextKey string

const clientKey = contextKey("mqtt_client")

// Client options.
const (
	writeTimeout = 1 * time.Second
	pingTimeout  = 10 * time.Second
)

var (
	messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
		log.Info().Str("msg", string(msg.Payload())).Str("topic", msg.Topic()).Msg("Received a missed message")
	}

	connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
		log.Info().Msg("Client connected to broker")
	}

	connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
		log.Info().Err(err).Msg("Connection lost")
	}

	reconnectHandler mqtt.ReconnectHandler = func(mqtt.Client, *mqtt.ClientOptions) {
		log.Info().Msg("Attempting to reconnect")
	}
)

// ConfigOptions is config options for an MQTT client.
type ConfigOptions struct {
	Server   string
	ClientID string
	Username string
	Password string
}

func NewClient(ctx context.Context, config ConfigOptions) mqtt.Client {
	// Set global logger.
	setLogger(ctx)

	opts := mqtt.NewClientOptions()

	// The following optins are set in additions to package defaults.
	opts.AddBroker(config.Server)
	opts.SetClientID(config.ClientID + "-" + uuid.NewString())

	// Official suggestion: Unless ordered delivery of messages is essential (and you have configured your broker to support this e.g. max_inflight_messages=1 in mosquito) then set ClientOptions.SetOrderMatters(false). Doing so will avoid the below issue (deadlocks due to blocking message handlers).
	opts.SetOrderMatters(false)
	opts.SetCleanSession(false)
	opts.SetUsername(config.Username)
	opts.SetPassword(config.Password)
	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.OnConnectionLost = connectLostHandler
	opts.OnReconnecting = reconnectHandler
	opts.OnConnect = connectHandler

	opts.WriteTimeout = writeTimeout // Minimal delays on writes
	opts.PingTimeout = pingTimeout

	// Automate connection management (will keep trying to connect and will reconnect if network drops)
	opts.ConnectRetry = true

	return mqtt.NewClient(opts)
}

// setLogger sets a customized input logger for MQTT client from context.
// By this way, user can decide the log level.
func setLogger(ctx context.Context) {
	log.Logger = log.Ctx(ctx).With().Str("component", "mqtt-client").Logger()
}

// CheckConnectivity checks MQTT client connectivity with a timeout.
func CheckConnectivity(client mqtt.Client, timeout time.Duration) error {
	if token := client.Connect(); token.WaitTimeout(timeout) && token.Error() != nil {
		return token.Error()
	}
	return nil
}

// WithContext creates a new MQTT client with provided client attached.
func WithContext(ctx context.Context, client mqtt.Client) context.Context {
	return context.WithValue(ctx, clientKey, client)
}

// FromContext returns the MQTT client stored in context. If no such client exists, it returns nil.
func FromContext(ctx context.Context) mqtt.Client {
	if client, ok := ctx.Value(clientKey).(mqtt.Client); ok {
		return client
	}
	return nil
}
