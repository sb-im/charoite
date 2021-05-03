package turn

import (
	"fmt"
	"net"
	"strconv"

	"github.com/pion/turn/v2"
	"github.com/rs/zerolog"
)

func Serve(logger *zerolog.Logger, cfg *ConfigOptions) (*turn.Server, error) {
	udpListener, err := net.ListenPacket("udp4", "0.0.0.0:"+strconv.Itoa(cfg.Port))
	if err != nil {
		return nil, fmt.Errorf("could not create udp4 listener: %w", err)
	}
	logger.Info().Str("host", "0.0.0.0").Int("port", cfg.Port).Msg("created udp4 listener")

	// Cache users for easy lookup later
	// If passwords are stored they should be saved to your DB hashed using turn.GenerateAuthKey
	usersMap := make(map[string][]byte)
	usersMap[cfg.Username] = turn.GenerateAuthKey(cfg.Username, cfg.Realm, cfg.Password)

	s, err := turn.NewServer(turn.ServerConfig{
		LoggerFactory: adapter(&pionLogger{logger}),
		Realm:         cfg.Realm,
		AuthHandler: func(username, realm string, srcAddr net.Addr) (key []byte, ok bool) {
			if key, ok := usersMap[username]; ok {
				return key, true
			}
			return nil, false
		},
		PacketConnConfigs: []turn.PacketConnConfig{
			{
				PacketConn: udpListener,
				RelayAddressGenerator: &turn.RelayAddressGeneratorPortRange{
					RelayAddress: net.ParseIP(cfg.PublicIP), // Claim that we are listening on IP passed by user (This should be your Public IP)
					Address:      "0.0.0.0",                 // But actually be listening on every interface
					MinPort:      uint16(cfg.RelayMinPort),
					MaxPort:      uint16(cfg.RelayMaxPort),
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("could not create TURN server: %w", err)
	}
	logger.Info().
		Uint("min_port", cfg.RelayMinPort).
		Uint("max_port", cfg.RelayMaxPort).
		Str("public_ip", cfg.PublicIP).
		Msg("started turn server")

	return s, nil
}
