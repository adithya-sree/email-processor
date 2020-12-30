package main

import (
	"github.com/adithya-sree/logger"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"os"
	"os/signal"
)

const (
	logFile               = "/var/log/email-processor.log"

	mqttBroker            = "tcp://192.168.0.116:1883"
	mqttClient            = "email-processor"
	mqttTopic             = "/emails"
	mqttAutoReconnect     = true
	mqttDisconnectTimeout = 5000

	smtpHost = "smtp.gmail.com"
	smtpPort = 587
	from     = ""
	password = ""
)

var out = logger.GetLogger(logFile, "main")

func main() {
	out.Info("Starting Email Processor")
	// Initialize MQTT Client
	out.Info("Connecting to Broker")
	mqttClient, err := initializeMqttClient(mqttBroker, mqttClient, mqttAutoReconnect)
	if err != nil {
		out.Errorf("Error connecting to client: [%v]", err)
		os.Exit(2)
	}
	out.Info("Connected to Broker")
	// Initialize SMTP Client
	out.Info("Connecting to SMTP server")
	smtpClient, err := initializeSmtpClient(smtpHost, from, password, smtpPort)
	if err != nil {
		out.Errorf("Error connecting to client: [%v]", err)
		os.Exit(2)
	}
	out.Info("Connected to SMTP server")
	// Subscribe to topic
	mqttClient.Subscribe(mqttTopic, 1, handleMessage(smtpClient))
	// Create Channel for Safe Exit
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	// Run Blocking
	<-c
	exit(mqttClient, smtpClient)
}

func exit(mqttClient mqtt.Client, smtpClient *smtpClient) {
	// Exit
	out.Info("Process is existing, closing connections...")
	// Disconnect from Broker
	mqttClient.Unsubscribe(mqttTopic)
	mqttClient.Disconnect(mqttDisconnectTimeout)
	// Disconnect from SMTP server
	err := smtpClient.client.Close()
	if err != nil {
		out.Errorf("Error while existing process: [%v]", err)
	}
	out.Info("Connections closed, exited")
	os.Exit(0)
}