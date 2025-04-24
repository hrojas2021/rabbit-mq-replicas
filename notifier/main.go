package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/streadway/amqp"
)

type Message struct {
	Body string `json:"body"`
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func connectRabbitMQ() *amqp.Connection {
	var conn *amqp.Connection
	var err error

	for i := 0; i < 10; i++ {
		conn, err = amqp.Dial("amqp://guest:guest@rabbitmq:5672/")
		if err == nil {
			return conn
		}
		log.Printf("Attempt %d: Waiting for RabbitMQ... (%s)", i+1, err)
		time.Sleep(5 * time.Second)
	}

	log.Fatalf("Could not connect to RabbitMQ after retries: %s", err)
	return nil
}

func main() {
	conn := connectRabbitMQ()
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	queueName := "notifier_queue"
	q, err := ch.QueueDeclare(
		queueName,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	failOnError(err, "Failed to declare queue")

	err = ch.QueueBind(
		q.Name,
		"",                   //empty its fanout
		"broadcast_exchange", // exchange
		false,
		nil,
	)
	failOnError(err, "Failed to bind queue to exchange")

	err = ch.Qos(
		1,
		0,
		false,
	)
	failOnError(err, "Failed to set QoS")

	msgs, err := ch.Consume(
		q.Name,
		"",
		false,
		false,
		false,
		false,
		nil, // arguments
	)
	failOnError(err, "Failed to register a consumer")

	notifierID := os.Getenv("NOTIFIER_ID")
	if notifierID == "" {
		notifierID = "notifier-unknown"
	}

	log.Printf("Notifier %s is running and waiting for messages...", notifierID)

	logPath := "/shared/logs/messages.log"

	for d := range msgs {
		var msg Message
		json.Unmarshal(d.Body, &msg)

		now := time.Now().Format(time.RFC3339)
		entry := fmt.Sprintf("[%s] [%s] Received message: %s\n", now, notifierID, msg.Body)

		f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			f.WriteString(entry)
			f.Close()
			d.Ack(false)
		} else {
			log.Printf("Failed to write to log file: %v", err)
		}
	}
}
