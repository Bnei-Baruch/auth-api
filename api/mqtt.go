package api

import (
	"fmt"
	"github.com/eclipse/paho.mqtt.golang"
	"strings"
)

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
	//fmt.Printf("Received message: %s from topic: %s\n", m.Payload(), m.Topic())
	id := "false"
	s := strings.Split(m.Topic(), "/")
	p := string(m.Payload())

	if s[0] == "kli" && len(s) == 5 {
		id = s[4]
	} else if s[0] == "exec" && len(s) == 4 {
		id = s[3]
	}

	if id == "false" {
		//switch p {
		//case "start":
		//	go a.startExecMqtt(ep)
		//case "stop":
		//	go a.stopExecMqtt(ep)
		//case "status":
		//	go a.execStatusMqtt(ep)
		//}
	}

	if id != "false" {
		//switch p {
		//case "start":
		//	go a.startExecMqttByID(ep, id)
		//case "stop":
		//	go a.stopExecMqttByID(ep, id)
		//case "status":
		//	go a.execStatusMqttByID(ep, id)
		//case "cmdstat":
		//	go a.cmdStatMqtt(ep, id)
		//case "progress":
		//	go a.getProgressMqtt(ep, id)
		//case "report":
		//	go a.getReportMqtt(ep, id)
		//case "alive":
		//	go a.isAliveMqtt(ep, id)
		//}
	}
}

func (a *App) Publish(topic string, message string) {
	text := fmt.Sprintf(message)
	//a.Msg.Publish(topic, byte(1), false, text)
	if token := a.Msg.Publish(topic, byte(1), false, text); token.Wait() && token.Error() != nil {
		fmt.Printf("Publish message error: %s\n", token.Error())
	}
}
