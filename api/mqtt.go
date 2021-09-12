package api

import (
	"encoding/json"
	"fmt"
	"github.com/eclipse/paho.mqtt.golang"
	"strings"
)

type Message struct {
	User User   `json:"user,omitempty"`
	Type string `json:"type,omitempty"`
	Text string `json:"text,omitempty"`
}

type User struct {
	ID      string `json:"id,omitempty"`
	Role    string `json:"role,omitempty"`
	Display string `json:"display,omitempty"`
}

func (a *App) SubMQTT(c mqtt.Client) {
	fmt.Println("- Connected to MQTT -")
	if token := a.Msg.Subscribe("auth/service/#", byte(1), a.MsgHandler); token.Wait() && token.Error() != nil {
		fmt.Printf("MQTT Subscription error: %s\n", token.Error())
	} else {
		fmt.Printf("%s\n", "MQTT Subscription: auth/service/#")
	}
}

func (a *App) LostMQTT(c mqtt.Client, e error) {
	fmt.Printf("MQTT Connection Error: %s\n", e)
}

func (a *App) MsgHandler(c mqtt.Client, m mqtt.Message) {
	id := "false"
	s := strings.Split(m.Topic(), "/")
	//p := string(m.Payload())

	if s[0] == "kli" && len(s) == 5 {
		id = s[4]
	} else if s[0] == "exec" && len(s) == 4 {
		id = s[3]
	}

	fmt.Printf("Received message: %s from topic: %s %s\n", m.Payload(), m.Topic(), id)
}

func (a *App) Publish(topic string, message string) {
	text := fmt.Sprintf(message)
	//a.Msg.Publish(topic, byte(1), false, text)
	if token := a.Msg.Publish(topic, byte(1), false, text); token.Wait() && token.Error() != nil {
		fmt.Printf("Publish message error: %s\n", token.Error())
	}
}

func (a *App) SendMessage(id string) {
	topic := "galaxy/users/" + id
	m := &Message{
		Text: "You approved to Arvut System! Please ReLogin to System with updated permission.",
		Type: "client-chat",
		User: User{
			ID:      id,
			Role:    "service",
			Display: "Auth System",
		},
	}

	message, err := json.Marshal(m)
	if err != nil {
		fmt.Printf("Message parsing: %s\n", err)
	}

	text := fmt.Sprintf(string(message))
	if token := a.Msg.Publish(topic, byte(1), true, text); token.Wait() && token.Error() != nil {
		fmt.Printf("Send message error: %s\n", token.Error())
	}
}
