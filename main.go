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
	smtpHost = "smtp.gmail.com"
	smtpPort = "587"
	logFile  = "/var/log/email-processor.log"
	from     = ""
	password = ""
)

var (
	out           = logger.GetLogger(logFile, "main")
	auth          = smtp.PlainAuth("", from, password, smtpHost)

	clientOptions = ClientOptions{
		mqttProtocol:          "tcp",
		mqttBrokerIp:          "192.168.0.116",
		mqttPort:              "1883",
		mqttClient:            "email-processor",
		mqttTopic:             "/emails",
		mqttAutoReconnect:     true,
		mqttDisconnectTimeout: 5000,
	}
)

type ClientOptions struct {
	mqttProtocol          string
	mqttBrokerIp          string
	mqttPort              string
	mqttClient            string
	mqttTopic             string
	mqttAutoReconnect	  bool
	mqttDisconnectTimeout uint
}

type Email struct {
	Title string   `json:"title"`
	Body  string   `json:"body"`
	To    []string `json:"to"`
}

func main() {
	out.Info("Starting Email Processor")
	// Initialize MQTT Client
	out.Info("Connecting to Broker")
	client, err := initializeClient(clientOptions)
	if err != nil {
		out.Errorf("Error connecting to client: [%v]", err)
		os.Exit(2)
	}
	out.Info("Connected to Broker")
	// Subscribe to topic
	client.Subscribe(clientOptions.mqttTopic, 1, handleMessage)
	// Create Channel for Safe Exit
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	// Run Blocking
	<-c
	// Disconnect from Broker
	client.Unsubscribe(clientOptions.mqttTopic)
	client.Disconnect(clientOptions.mqttDisconnectTimeout)
	// Exit Gracefully
	out.Info("Process is exiting")
	os.Exit(0)
}

func initializeClient(options ClientOptions) (mqtt.Client, error) {
	// Initialize Client Options
	opts := &mqtt.ClientOptions{}
	opts.SetClientID(options.mqttClient)
	opts.SetAutoReconnect(options.mqttAutoReconnect)
	opts.AddBroker(fmt.Sprintf("%s://%s:%s", options.mqttProtocol, options.mqttBrokerIp, options.mqttPort))
	// Initialize Client
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}
	return client, nil
}

func handleMessage(_ mqtt.Client, message mqtt.Message) {
	out.Infof("Received message on topic: [%v]", clientOptions.mqttTopic)
	// Unmarshal Message
	var mail Email
	err := json.Unmarshal(message.Payload(), &mail)
	if err != nil {
		out.Warnf("Unable to parse message: [%v]", err)
		return
	}
	out.Infof("Parsed message: [%v]", mail)
	// Send Mail
	err = processMail(mail)
	if err != nil {
		out.Errorf("Error sending email: [%v]", err)
		return
	}
	out.Info("Processed message over SMTP")
}

func processMail(mail Email) error {
	// Format Message
	msg := []byte(fmt.Sprintf("Subject: %s\r\n\r\n%s\r\n", mail.Title, mail.Body))
	// Format Connection
	conn := fmt.Sprintf("%s:%s", smtpHost, smtpPort)
	// Send Email
	return smtp.SendMail(conn, auth, from, mail.To, msg)
}