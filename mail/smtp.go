package mail

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/smtp"
	"sync"
	"time"
)

type Email struct {
	Title string   `json:"title"`
	Body  string   `json:"body"`
	To    []string `json:"to"`
}

type SmtpConnectionPool struct {
	mu             sync.Mutex
	connectionPool []*smtp.Client
	timeout        int
}

type smtpResponse struct {
	client *smtp.Client
	error  error
}

// Gets a single connection
func (pool *SmtpConnectionPool) GetConnection() (*smtp.Client, error) {
	// Default Timeout
	timer := time.NewTimer(time.Duration(pool.timeout) * time.Millisecond)
	cancel := make(chan int, 1)
	// Waits to either receive a connection or to timeout
	select {
	// if timeout is reached, send cancel signal
	// on cancel channel and return error
	case <- timer.C:
		cancel <- 0
		return nil, errors.New("timed out waiting for connection")
	// If connection is received, return connection from pool
	case conn := <- getConnection(pool, cancel):
		return conn, nil
	}
}

// Return connection back to pool
func (pool *SmtpConnectionPool) CloseConnection() {
	// Mutex Lock
	pool.mu.Lock()
	// Close all connections in pool
	for _, conn := range pool.connectionPool {
		_ = conn.Close()
	}
	pool.mu.Unlock()
}

// Close all connections in pool
func (pool *SmtpConnectionPool) ReturnConnection(conn *smtp.Client) {
	// Mutex Lock
	pool.mu.Lock()
	// Return connection to pool
	pool.connectionPool = append(pool.connectionPool, conn)
	pool.mu.Unlock()
}

// Initializes SMTP connection pool
func InitializeSmtpConnectionPool(numConnections, timeout int, host, user, pass string, port uint) (*SmtpConnectionPool, error) {
	// List of initial connections
	var connections []*smtp.Client
	// Connections channel
	c := initializeConnections(numConnections, host, user, pass, port)
	for i := 0; i < numConnections; i++ {
		// Wait for each connection to return
		if smtpResponse := <- c; smtpResponse.error == nil {
			// Add connection to pool
			connections = append(connections, smtpResponse.client)
		} else {
			return nil, smtpResponse.error
		}
	}
	// Return connection pool
	return &SmtpConnectionPool{
		mu:             sync.Mutex{},
		connectionPool: connections,
		timeout:        timeout,
	}, nil
}

func initializeConnections(len int, host, user, pass string, port uint) <-chan *smtpResponse {
	// Create response channel
	c := make(chan *smtpResponse, len)
	for i := 0; i < len; i++ {
		// Create individual connections
		go initializeConnection(c, host, user, pass, port)
	}
	return c
}

func initializeConnection(c chan *smtpResponse, host, user, pass string, port uint) {
	// Initialize SMTP Client connection
	smtpClient, err := initializeSmtpClient(host, user, pass, port)
	c <- &smtpResponse{
		client: smtpClient,
		error:  err,
	}
}

func initializeSmtpClient(host, user, pass string, port uint) (*smtp.Client, error) {
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
	return client, nil
}

func getConnection(pool *SmtpConnectionPool, cancel chan int) <-chan *smtp.Client {
	// Create client channel
	c := make(chan *smtp.Client, 1)
	go waitForConnection(pool, cancel, c)
	// Return client channel
	return c
}

func waitForConnection(pool *SmtpConnectionPool, cancel chan int, response chan *smtp.Client) {
	exit := false
	// Iterate over channel selects until a connection
	// is found or the timeout is reached
	for {
		select {
		// If cancel channel received signal exit iteration
		case <- cancel:
			exit = true
		// If no receive operation is available on channel,
		// attempt to get connection from pool
		default:
			// Lock Mutex
			pool.mu.Lock()
			if len(pool.connectionPool) >= 1 {
				// If connection exists, return connection on channel
				response <- pool.connectionPool[0]
				// Remove connection from pool
				pool.connectionPool = remove(pool.connectionPool, 0)
				// Exit over iteration
				exit = true
			}
			// Unlock Mutex
			pool.mu.Unlock()
		}
		// If timeout is reached or connection is found
		// break iteration
		if exit {
			break
		}
	}
}

func remove(a []*smtp.Client, i int) []*smtp.Client {
	return append(a[:i], a[i+1:]...)
}