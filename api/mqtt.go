package api

import (
	"encoding/json"
	"fmt"
	"github.com/eclipse/paho.golang/paho"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"log"
	"os"
	"time"
)

type Request struct {
	Function string `json:"function"`
	Param1   int    `json:"param1"`
	Param2   int    `json:"param2"`
}

func ClientMQTT() {
	server := os.Getenv("MQTT_URL")
	username := os.Getenv("MQTT_USER")
	password := os.Getenv("MQTT_PASS")

	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("ssl://%s", server))
	opts.SetClientID("go_mqtt_client")
	opts.SetUsername(username)
	opts.SetPassword(password)
	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
}

func onMessage(p *paho.Publish) {
	log.Printf("Got message: %v", p.String())
	var pMsg map[string]interface{}
	if err := json.Unmarshal([]byte(string(p.Payload)), &pMsg); err != nil {
		log.Printf("Failed to decode Request: %v", pMsg)
	}
	log.Printf("Decoded message: %v", pMsg)
}

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	fmt.Printf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	fmt.Println("Connected")
	sub(client)
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	fmt.Printf("Connect lost: %v", err)
}

func publish(client mqtt.Client) {
	num := 10
	for i := 0; i < num; i++ {
		text := fmt.Sprintf("Message %d", i)
		token := client.Publish("topic/test", 0, false, text)
		token.Wait()
		time.Sleep(time.Second)
	}
}

func sub(client mqtt.Client) {
	topic := "galaxy/room/1051"
	token := client.Subscribe(topic, 2, nil)
	token.Wait()
	fmt.Printf("Subscribed to topic: %s", topic)
}
