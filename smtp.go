package main

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"sync"
)

type email struct {
	Title string   `json:"title"`
	Body  string   `json:"body"`
	To    []string `json:"to"`
}

type smtpClient struct {
	client *smtp.Client
	mu     sync.Mutex
}

func initializeSmtpClient(host, user, pass string, port uint) (*smtpClient, error) {
	// Connect to SMTP server
	client, err := smtp.Dial(fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return nil, err
	}
	// Skip Verify
	if err = client.StartTLS(&tls.Config{InsecureSkipVerify: true}); err != nil {
		return nil, err
	}
	// Authenticate
	if err = client.Auth(smtp.PlainAuth("", user, pass, host)); err != nil {
		return nil, err
	}
	return &smtpClient{
		client: client,
		mu:     sync.Mutex{},
	}, nil
}

func sendMail(c *smtpClient, mail email) error {
	c.mu.Lock()
	msg := []byte(fmt.Sprintf("Subject: %s\r\n\r\n%s\r\n", mail.Title, mail.Body))
	if err := c.client.Mail(from); err != nil {
		c.mu.Unlock()
		return err
	}
	for _, addr := range mail.To {
		if err := c.client.Rcpt(addr); err != nil {
			c.mu.Unlock()
			return err
		}
	}
	w, err := c.client.Data()
	if err != nil {
		c.mu.Unlock()
		return err
	}
	_, err = w.Write(msg)
	if err != nil {
		c.mu.Unlock()
		return err
	}
	err = w.Close()
	if err != nil {
		c.mu.Unlock()
		return err
	}
	c.mu.Unlock()
	return nil
}