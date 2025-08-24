package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/streadway/amqp"
)

// Notification represents a single notification message
type Notification struct {
	ID      string `json:"id"`
	UserID  string `json:"user_id"`
	Message string `json:"message"`
}

// NotificationQueue represents a message queue for notifications
type NotificationQueue struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	queue   amqp.Queue
}

// NewNotificationQueue creates a new notification queue
func NewNotificationQueue(amqpURL string) (*NotificationQueue, error) {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return nil, err
	}
	channel, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	queue, err := channel.QueueDeclare(
		"notifications", // name
		true,         // durable
		false,       // delete when unused
		false,       // exclusive
		false,       // no-wait
		nil,         // arguments
	)
	if err != nil {
		return nil, err
	}
	return &NotificationQueue{
		conn:    conn,
		channel: channel,
		queue:   queue,
	}, nil
}

// Publish publishes a notification to the queue
func (q *NotificationQueue) Publish(n *Notification) error {
	body, err := json.Marshal(n)
	if err != nil {
		return err
	}
	return q.channel.Publish(
		"",         // exchange
		q.queue.Name, // routing key
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
}

// WebSocketHub represents a WebSocket hub for real-time notifications
type WebSocketHub struct {
	// Registered clients.
	clients    map[*Client]bool
	broadcast  chan *Notification
	register   chan *Client
	unregister chan *Client
}

// NewWebSocketHub creates a new WebSocket hub
func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		broadcast:  make(chan *Notification),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

// Client represents a single WebSocket client
type Client struct {
	hub  *WebSocketHub
	conn *websocket.Conn
	send chan []byte
}

// NewClient creates a new WebSocket client
func NewClient(hub *WebSocketHub, conn *websocket.Conn) *Client {
	return &Client{
		hub:  hub,
		conn: conn,
		send: make(chan []byte, 256),
	}
}

// Run starts the WebSocket client
func (c *Client) Run() {
	go c.writePump()
	go c.readPump()
}

// writePump pumps messages from the hub to the client
func (c *Client) writePump() {
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.conn.WriteMessage(websocket.TextMessage, message)
		}
	}
}

// readPump pumps messages from the client to the hub
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
		// Handle incoming message from client (e.g., subscribe/unsubscribe)
		// For simplicity, we'll ignore this for now
	}
}

// Run starts the WebSocket hub
func (h *WebSocketHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case notification := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- json.Marshal(notification):
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

func main() {
	amqpURL := "amqp://guest:guest@localhost:5672/"
	queue, err := NewNotificationQueue(amqpURL)
	if err != nil {
		log.Fatal(err)
	}

	hub := NewWebSocketHub()
	go hub.Run()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Upgrade(w, r, nil, 1024, 1024)
		if err != nil {
			log.Println(err)
			return
		}
		client := NewClient(hub, conn)
		hub.register <- client

		go client.Run()
	})

	http.HandleFunc("/publish", func(w http.ResponseWriter, r *http.Request) {
		var notification Notification
		err := json.NewDecoder(r.Body).Decode(&notification)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err = queue.Publish(&notification)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	})

	go func() {
		msgs, err := queue.channel.Consume(
			queue.queue.Name, // queue
			"",         // consumer
			true,       // auto-ack
			false,      // exclusive
			false,      // no-local
			false,      // no-wait
			nil,         // args
		)
		if err != nil {
			log.Fatal(err)
		}
		for msg := range msgs {
			var notification Notification
			err := json.Unmarshal(msg.Body, &notification)
			if err != nil {
				log.Println(err)
				continue
			}
			hub.broadcast <- &notification
		}
	}()

	log.Println("Server started on port 8080")
	http.ListenAndServe(":8080", nil)
}