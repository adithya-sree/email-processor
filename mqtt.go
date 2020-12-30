package main

import (
	"encoding/json"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
)

func initializeMqttClient(broker, clientId string, reconnect bool) (mqtt.Client, error) {
	// Initialize Client Options
	opts := &mqtt.ClientOptions{}
	opts.SetClientID(clientId)
	opts.SetAutoReconnect(reconnect)
	opts.AddBroker(broker)
	// Initialize Client
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}
	return client, nil
}

func handleMessage(smtpClient *smtpClient) func (mqtt.Client, mqtt.Message) {
	return func(client mqtt.Client, message mqtt.Message) {
		// Handle Message
		sessionId := uuid.New().String()
		out.Infof("[%s] - Received Message on Topic", sessionId)
		// Unmarshal Message
		var mail email
		if err := json.Unmarshal(message.Payload(), &mail); err != nil {
			out.Warnf("[%s] - Unable to unmarshal message: [%v]", sessionId, err)
			return
		}
		// Send Mail
		if err := sendMail(smtpClient, mail); err != nil {
			out.Errorf("[%s] - Error sending mail: [%v]", sessionId, err)
			return
		}
		// Log Success
		out.Infof("[%s] - Processed mail to SMTP server", sessionId)
	}
}