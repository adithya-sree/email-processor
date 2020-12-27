package main

import (
	"encoding/json"
	"fmt"
	"github.com/adithya-sree/logger"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"net/smtp"
	"os"
	"os/signal"
)

const (
	smtpHost          = "smtp.gmail.com"
	smtpPort          = "587"
	logFile           = "/var/log/email-processor.log"
	from              = ""
	password          = ""

	mqttProtocol      = "tcp"
	mqttBrokerIp      = "192.168.0.116"
	mqttPort          = "1883"
	mqttClient        = "email-processor"
	mqttTopic         = "/emails"
	mqttAutoReconnect = true
)

type Email struct {
	Title string   `json:"title"`
	Body  string   `json:"body"`
	To    []string `json:"to"`
}

var (
	out    = logger.GetLogger(logFile, "main")
	auth   = smtp.PlainAuth("", from, password, smtpHost)
	client = &mqtt.ClientOptions{}
)

func init() {
	client.SetClientID(mqttClient)
	client.SetAutoReconnect(mqttAutoReconnect)
	client.AddBroker(fmt.Sprintf("%s://%s:%s", mqttProtocol, mqttBrokerIp, mqttPort))
}

func main() {
	out.Info("Starting Email Processor")
	// Initialize MQTT Client
	out.Info("Connecting to Broker")
	client := mqtt.NewClient(client)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		out.Errorf("Error connecting to client: [%v]", token.Error())
		return
	}
	out.Info("Connected to Broker")
	// Subscribe to topic
	client.Subscribe(mqttTopic, 1, func(client mqtt.Client, message mqtt.Message) {
		// Handle Message on Topic
		payload := message.Payload()
		out.Infof("Received Message on Topic: [%s]", payload)
		// Parse Message
		var mail Email
		err := json.Unmarshal(payload, &mail)
		if err != nil {
			out.Warnf("Unable to unmarshal message: [%v]", err)
			return
		}
		// Format Message
		msg := []byte(fmt.Sprintf("Subject: %s\r\n\r\n%s\r\n", mail.Title, mail.Body))
		// Send Email
		err = smtp.SendMail(smtpHost+":"+smtpPort, auth, from, mail.To, msg)
		if err != nil {
			out.Errorf("Error sending email: [%v]", err)
			return
		}
		out.Info("Message sent over SMTP")
	})
	// Create Channel for Safe Exit
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	// Run Blocking
	<-c
	// Exit Gracefully
	out.Info("Process is exiting")
	os.Exit(0)
}