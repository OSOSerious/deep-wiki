package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/streadway/amqp"
)

// Notification struct
type Notification struct {
	UserID  string `json:"user_id"`
	Message string `json:"message"`
}

// WebSocketConnection struct
type WebSocketConnection struct {
	conn *websocket.Conn
}

// RabbitMQConnection struct
type RabbitMQConnection struct {
	conn *amqp.Connection
}

// NewRabbitMQConnection creates a new RabbitMQ connection
func NewRabbitMQConnection(amqpURI string) (*RabbitMQConnection, error) {
	conn, err := amqp.Dial(amqpURI)
	if err != nil {
		return nil, err
	}
	return &RabbitMQConnection{conn: conn}, nil
}

// PublishNotification publishes a notification to RabbitMQ
func (r *RabbitMQConnection) PublishNotification(notification Notification) error {
	ch, err := r.conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	body, err := json.Marshal(notification)
	if err != nil {
		return err
	}

	err = ch.Publish(
		"",         // exchange
		"notifications", // routing key
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			ContentType: "text/json",
			Body:       body,
		})
	if err != nil {
		return err
	}
	return nil
}

// NewWebSocketConnection creates a new WebSocket connection
func NewWebSocketConnection(w http.ResponseWriter, r *http.Request) (*WebSocketConnection, error) {
	upgrader := websocket.Upgrader{}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}
	return &WebSocketConnection{conn: conn}, nil
}

// ReceiveNotification receives a notification from RabbitMQ
func (w *WebSocketConnection) ReceiveNotification(r *RabbitMQConnection) {
	ch, err := r.conn.Channel()
	if err != nil {
		log.Println(err)
		return
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"notifications", // name
		true,           // durable
		false,          // delete when unused
		false,          // exclusive
		false,          // noWait
		nil,            // arguments
	)
	if err != nil {
		log.Println(err)
		return
	}

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		log.Println(err)
		return
	}

	for msg := range msgs {
		var notification Notification
		err := json.Unmarshal(msg.Body, &notification)
		if err != nil {
			log.Println(err)
			continue
		}

		err = w.conn.WriteJSON(notification)
		if err != nil {
			log.Println(err)
		}
	}
}

func main() {
	amqpURI := "amqp://guest:guest@localhost:5672/"
	rabbitMQConn, err := NewRabbitMQConnection(amqpURI)
	if err != nil {
		log.Fatal(err)
	}
	defer rabbitMQConn.conn.Close()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		wsConn, err := NewWebSocketConnection(w, r)
		if err != nil {
			log.Println(err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		defer wsConn.conn.Close()

		wsConn.ReceiveNotification(rabbitMQConn)
	})

	log.Println("Server started on port 8080")
	http.ListenAndServe(":8080", nil)
}