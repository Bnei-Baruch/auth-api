package api

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net"
	"os"
	"time"

	"github.com/eclipse/paho.golang/paho"
	pkgerr "github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type MQTTListener struct {
	client    *paho.Client
	connected bool
}

func NewMQTTListener() *MQTTListener {
	return &MQTTListener{connected: false}
}

func (l *MQTTListener) Start() error {
	l.client = paho.NewClient(paho.ClientConfig{
		ClientID: "go_client_id",
		OnConnectionLost: func(err error) {
			l.connected = false
			log.Warn().Msgf("MQTT OnConnectionLost: %+v", err)
			for !l.connected {
				time.Sleep(10 * time.Second)
				if err = l.init(); err != nil {
					log.Error().Err(err).Msg("error initializing mqtt on connection lost")
				}
			}
		},
	})

	//l.client.SetErrorLogger(NewPahoLogAdapter(zerolog.ErrorLevel))
	//debugLog := NewPahoLogAdapter(zerolog.InfoLevel)
	//l.client.SetDebugLogger(debugLog)
	//l.client.PingHandler.SetDebug(debugLog)
	//l.client.Router.SetDebug(debugLog)

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

	var sessionExpiryInterval = uint32(5)

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

	l.connected = true

	sa, err := l.client.Subscribe(context.Background(), &paho.Subscribe{
		Subscriptions: map[string]paho.SubscribeOptions{
			"auth/service": {QoS: byte(2), NoLocal: true},
		},
	})

	if err != nil {
		return pkgerr.Wrap(err, "client.Subscribe")
	}
	if sa.Reasons[0] != byte(2) {
		return pkgerr.Errorf("MQTT subscribe error: %d ", sa.Reasons[0])
	}

	l.client.Router.RegisterHandler("auth/service", l.OnMessage)

	err = l.SendMessage(`{"message":"test"}`)
	if err != nil {
		return pkgerr.Wrap(err, "client.Publish")
	}

	return nil
}

func (l *MQTTListener) Close() error {
	if err := l.client.Disconnect(&paho.Disconnect{ReasonCode: 0}); err != nil {
		return pkgerr.Wrap(err, "client.Disconnect")
	}
	return nil
}

func (l *MQTTListener) OnMessage(p *paho.Publish) {
	log.Info().Str("payload", p.String()).Msg("MQTT OnMessage")
	if err := HandleMessage(string(p.Payload)); err != nil {
		log.Error().Err(err).Msg("OnMessage error")
	}
}

func (l *MQTTListener) SendMessage(msg string) error {

	pa, err := l.client.Publish(context.Background(), &paho.Publish{
		QoS:        byte(2),
		Retain:     false,
		Topic:      "auth/service",
		Properties: &paho.PublishProperties{},
		Payload:    []byte(msg),
	})

	if err != nil {
		return pkgerr.Wrap(err, "client.Publish")
	}
	if pa.ReasonCode != 0 {
		return pkgerr.Errorf("MQTT Publish error: %d - %s", pa.ReasonCode, pa.Properties.ReasonString)
	}

	return nil
}

//type PahoLogAdapter struct {
//	level zerolog.Level
//}
//
//func NewPahoLogAdapter(level zerolog.Level) *PahoLogAdapter {
//	return &PahoLogAdapter{level: level}
//}
//
//func (a *PahoLogAdapter) Println(v ...interface{}) {
//	log.WithLevel(a.level).Msgf("mqtt: %s", fmt.Sprint(v...))
//}
//
//func (a *PahoLogAdapter) Printf(format string, v ...interface{}) {
//	log.WithLevel(a.level).Msgf("mqtt: %s", fmt.Sprintf(format, v...))
//}

func HandleMessage(payload string) error {
	var pMsg map[string]interface{}
	if err := json.Unmarshal([]byte(payload), &pMsg); err != nil {
		return pkgerr.Errorf("json.Unmarshal: %s", err)
	}

	return nil
}
