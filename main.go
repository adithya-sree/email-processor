package main

import (
	"github.com/adithya-sree/email-processor/mail"
	"github.com/adithya-sree/logger"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"os"
	"os/signal"
)

const (
	logFile = "/var/log/email-processor.log"

	mqttBroker            = "tcp://192.168.0.116:1883"
	mqttClient            = "email-processor"
	mqttTopic             = "/emails"
	mqttAutoReconnect     = true
	mqttDisconnectTimeout = 5000

	smtpHost              = "smtp.gmail.com"
	smtpPort              = 587
	smtpMaxConnection     = 10
	smtpTimeout           = 5000

	from                  = ""
	password              = ""
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
	connectionPool, err := mail.InitializeSmtpConnectionPool(smtpMaxConnection, smtpTimeout, smtpHost, from, password, smtpPort)
	if err != nil {
		out.Errorf("Error initializing connection pool: [%v]", err)
		os.Exit(2)
	}
	// Subscribe to topic
	mqttClient.Subscribe(mqttTopic, 1, handleMessage(connectionPool))
	// Create Channel for Safe Exit
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	// Run Blocking
	<-c
	// Exit
	exit(mqttClient, connectionPool)
}

func exit(mqttClient mqtt.Client, connectionPool *mail.SmtpConnectionPool) {
	// Exit
	out.Info("Process is existing, closing connections...")
	// Disconnect from Broker
	mqttClient.Unsubscribe(mqttTopic)
	mqttClient.Disconnect(mqttDisconnectTimeout)
	out.Info("Connections closed, exited")
	os.Exit(0)
}
