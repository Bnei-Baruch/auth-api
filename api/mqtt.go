package api

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/eclipse/paho.golang/paho"
	"github.com/eclipse/paho.golang/paho/extensions/rpc"
	"log"
	"os"
)

type Request struct {
	Function string `json:"function"`
	Param1   int    `json:"param1"`
	Param2   int    `json:"param2"`
}

func ClientMQTT() {
	server := os.Getenv("MQTT_URL")
	topic := "galaxy/room/1051"
	username := os.Getenv("MQTT_USER")
	password := os.Getenv("MQTT_PASS")

	conn, err := tls.Dial("tcp", server, nil)

	if err != nil {
		log.Fatalf("Failed to connect to %s: %s", server, err)
	}

	c := paho.NewClient()
	c.Conn = conn
	c.Router.RegisterHandler("galaxy/room/1051", onMessage)

	cp := &paho.Connect{
		KeepAlive:  30,
		CleanStart: true,
		ClientID:   "listen1",
		Username:   username,
		Password:   []byte(password),
	}

	cp.UsernameFlag = true
	cp.PasswordFlag = true

	ca, err := c.Connect(context.Background(), cp)
	if err != nil {
		log.Fatalln(err)
	}
	if ca.ReasonCode != 0 {
		log.Fatalf("Failed to connect to %s : %d - %s", server, ca.ReasonCode, ca.Properties.ReasonString)
	}

	fmt.Printf("Connected to %s\n", server)

	_, err = c.Subscribe(context.Background(), &paho.Subscribe{
		Subscriptions: map[string]paho.SubscribeOptions{
			topic: paho.SubscribeOptions{QoS: 2},
		},
	})

	if err != nil {
		log.Fatalf("failed to subscribe: %s", err)
	}

	h, err := rpc.NewHandler(c)
	if err != nil {
		log.Fatal(err)
	}

	_, err = h.Request(&paho.Publish{
		Topic:   "galaxy/room/1051",
		Payload: []byte(`{"function":"mul", "param1": 10, "param2": 5}`),
	})
}

func onMessage(p *paho.Publish) {
	log.Printf("Got message: %v", p.String())
	var pMsg map[string]interface{}
	if err := json.Unmarshal([]byte(string(p.Payload)), &pMsg); err != nil {
		log.Printf("Failed to decode Request: %v", pMsg)
	}
	log.Printf("Decoded message: %v", pMsg)
}
