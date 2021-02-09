package api

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"os"

	"github.com/eclipse/paho.golang/paho"
	pkgerr "github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type MQTTListener struct {
	client *paho.Client
}

func NewMQTTListener() *MQTTListener {
	return &MQTTListener{}
}

func (l *MQTTListener) Start() error {
	l.client = paho.NewClient(paho.ClientConfig{
		ClientID: "go_client_id",
		OnConnectionLost: func(err error) {
			log.Warn().Msgf("MQTT OnConnectionLost: %+v", err)
			if err := l.init(); err != nil {
				log.Error().Err(err).Msg("error initializing mqtt on connection lost")
			}
		},
	})
	l.client.SetErrorLogger(NewPahoLogAdapter(zerolog.ErrorLevel))
	debugLog := NewPahoLogAdapter(zerolog.InfoLevel)
	l.client.SetDebugLogger(debugLog)
	l.client.PingHandler.SetDebug(debugLog)
	l.client.Router.SetDebug(debugLog)
	return l.init()
}

func (l *MQTTListener) init() error {
	log.Info().Msg("Initializing MQTT Listener")

	var conn net.Conn
	var err error
	if os.Getenv("MQTT_SSL") == "true" {
		conn, err = tls.Dial("tcp", os.Getenv("MQTT_URL"), nil)
	} else {
		conn, err = net.Dial("tcp", os.Getenv("MQTT_URL"))
	}
	if err != nil {
		return pkgerr.Wrap(err, "conn.Dial")
	}

	l.client.Conn = conn

	var sessionExpiryInterval = uint32(math.MaxUint32)

	cp := &paho.Connect{
		ClientID:   "auth_api_client",
		KeepAlive:  30,
		CleanStart: true,
		Properties: &paho.ConnectProperties{
			SessionExpiryInterval: &sessionExpiryInterval,
		},
	}

	pwd := os.Getenv("MQTT_PASS")

	if pwd != "" {
		cp.Username = "auth_api_user"
		cp.Password = []byte(pwd)
		cp.UsernameFlag = true
		cp.PasswordFlag = true
	}

	ca, err := l.client.Connect(context.Background(), cp)
	if err != nil {
		return pkgerr.Wrap(err, "client.Connect")
	}
	if ca.ReasonCode != 0 {
		return pkgerr.Errorf("MQTT connect error: %d - %s", ca.ReasonCode, ca.Properties.ReasonString)
	}

	l.client.Router.RegisterHandler("galaxy/service/#", l.HandleServiceProtocol)

	sa, err := l.client.Subscribe(context.Background(), &paho.Subscribe{
		Subscriptions: map[string]paho.SubscribeOptions{
			"galaxy/service/#": {QoS: byte(2)},
		},
	})
	if err != nil {
		return pkgerr.Wrap(err, "client.Subscribe")
	}
	if sa.Reasons[0] != byte(2) {
		return pkgerr.Errorf("MQTT subscribe error: %d ", sa.Reasons[0])
	}

	return nil
}

func (l *MQTTListener) Close() error {
	if err := l.client.Disconnect(&paho.Disconnect{ReasonCode: 0}); err != nil {
		return pkgerr.Wrap(err, "client.Disconnect")
	}
	return nil
}

func (l *MQTTListener) HandleServiceProtocol(p *paho.Publish) {
	log.Info().Str("payload", p.String()).Msg("MQTT handle service protocol")
	if err := HandleMessage(string(p.Payload)); err != nil {
		log.Error().Err(err).Msg("service protocol error")
	}
}

type PahoLogAdapter struct {
	level zerolog.Level
}

func NewPahoLogAdapter(level zerolog.Level) *PahoLogAdapter {
	return &PahoLogAdapter{level: level}
}

func (a *PahoLogAdapter) Println(v ...interface{}) {
	log.WithLevel(a.level).Msgf("mqtt: %s", fmt.Sprint(v...))
}

func (a *PahoLogAdapter) Printf(format string, v ...interface{}) {
	log.WithLevel(a.level).Msgf("mqtt: %s", fmt.Sprintf(format, v...))
}

func HandleMessage(payload string) error {
	var pMsg map[string]interface{}
	if err := json.Unmarshal([]byte(payload), &pMsg); err != nil {
		return pkgerr.Errorf("json.Unmarshal: %s", err)
	}

	return nil
}
