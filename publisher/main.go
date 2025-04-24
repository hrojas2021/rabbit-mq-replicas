package main

import (
	"encoding/json"
	"log"
	"net/http"
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

	err = ch.ExchangeDeclare(
		"broadcast_exchange", // name
		"fanout",             // type
		true,                 // durable
		false,                // auto-deleted
		false,                // internal
		false,                // no-wait
		nil,                  // arguments
	)
	failOnError(err, "Failed to declare a exchange")

	http.HandleFunc("/publish", func(w http.ResponseWriter, r *http.Request) {
		var msg Message
		err := json.NewDecoder(r.Body).Decode(&msg)
		if err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		body, _ := json.Marshal(msg)
		err = ch.Publish(
			"broadcast_exchange", "", false, false,
			amqp.Publishing{
				ContentType: "application/json",
				Body:        body,
			},
		)
		failOnError(err, "Failed to publish message")
		w.Write([]byte("Message published\n"))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Println("Listening on port", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
