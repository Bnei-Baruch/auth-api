package api

import (
	"fmt"
	"github.com/eclipse/paho.mqtt.golang"
	"os"
)

type MQTTListener struct {
	client mqtt.Client
}

func NewMQTTListener() *MQTTListener {
	return &MQTTListener{}
}

func (l *MQTTListener) init() error {
	server := os.Getenv("MQTT_URL")
	username := os.Getenv("MQTT_USER")
	password := os.Getenv("MQTT_PASS")

	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("ssl://%s", server))
	opts.SetClientID("auth_mqtt_client")
	opts.SetUsername(username)
	opts.SetPassword(password)
	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler
	l.client = mqtt.NewClient(opts)
	if token := l.client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	fmt.Printf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	fmt.Println("Connected")
	sub(client)
	publish(client)
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	fmt.Printf("Connect lost: %v", err)
}

func publish(client mqtt.Client) {
	text := fmt.Sprintf(`{"message":"test"}`)
	token := client.Publish("galaxy/service/shidur", 2, false, text)
	token.Wait()
}

func sub(client mqtt.Client) {
	topic := "galaxy/service/#"
	token := client.Subscribe(topic, 2, nil)
	token.Wait()
	fmt.Printf("Subscribed to topic: %s", topic)
}
