package main

import (
	"encoding/json"
	"fmt"
	"github.com/adithya-sree/email-processor/mail"
	"net/smtp"

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

func handleMessage(connectionPool *mail.SmtpConnectionPool) func(mqtt.Client, mqtt.Message) {
	return func(client mqtt.Client, message mqtt.Message) {
		// Handle Message
		sessionId := uuid.New().String()
		out.Infof("[%s] - Received Message on Topic", sessionId)
		// Unmarshal Message
		var m mail.Email
		if err := json.Unmarshal(message.Payload(), &m); err != nil {
			out.Warnf("[%s] - Unable to unmarshal message: [%v]", sessionId, err)
			return
		}
		// Get connection from pool
		smtpClient, err := connectionPool.GetConnection()
		if err != nil {
			out.Errorf("[%s] - Unable to get connection from pool: [%v]", sessionId, err)
			return
		}
		// Send Mail
		err = sendMail(smtpClient, m)
		// Return connection
		connectionPool.ReturnConnection(smtpClient)
		// Handle error
		if err != nil {
			out.Errorf("[%s] - Error sending mail: [%v]", sessionId, err)
			return
		}
		// Log Success
		out.Infof("[%s] - Processed mail to SMTP server", sessionId)
	}
}

func sendMail(c *smtp.Client, m mail.Email) error {
	// Format message
	msg := []byte(fmt.Sprintf("Subject: %s\r\n\r\n%s\r\n", m.Title, m.Body))
	// Issue mail command
	if err := c.Mail(from); err != nil {
		return err
	}
	// Issue rcpt command
	for _, addr := range m.To {
		if err := c.Rcpt(addr); err != nil {
			return err
		}
	}
	// Issue date command
	w, err := c.Data()
	if err != nil {
		return err
	}
	// Write mail
	_, err = w.Write(msg)
	if err != nil {
		return err
	}
	// Close connection
	err = w.Close()
	if err != nil {
		return err
	}
	return nil
}
