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
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("##### A panic is detected: #####", r)
		}
		ch.Close()
		fmt.Println("##### Closing the app:#####")
	}()

	q, err := ch.QueueDeclare(
		"demo_queue",
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	failOnError(err, "Failed to declare a queue")
	err = ch.Qos(
		1,     // prefetch count, this ensure each worker receives only 1 message at a time
		0,     // prefetchSize
		false, // global notes if QoS is global or just for the consumer
	)
	failOnError(err, "Failed to set QoS")

	msgs, err := ch.Consume(q.Name, "", false, false, false, false, nil)
	failOnError(err, "Failed to register a consumer")

	err = ch.QueueBind(
		q.Name,               // queue name
		"",                   // routing key
		"broadcast_exchange", // exchange
		false,
		nil,
	)
	failOnError(err, "Failed to register a consumer")

	logPath := "/shared/logs/messages.log"
	os.MkdirAll("/shared/logs", 0755)

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			var msg Message

			err = json.Unmarshal(d.Body, &msg)
			if err != nil {
				log.Println("Error reading the message:", err)
				d.Nack(false, true) // NACK message
			}
			now := time.Now().Format(time.RFC3339)
			workerID := os.Getenv("WORKER_ID")
			if workerID == "" {
				workerID = "unknown-worker"
			}

			if workerID == "worker-2" {
				log.Println("Failed to write to consume this message:", err)
				d.Nack(false, true) // simulate a DB error and nack the message
				break               // break the endless loop and  goroutine
				// The defer will close the remaining open resources
			}

			if workerID == "worker-3" {
				entry := fmt.Sprintf("#### [%s]HAS BROKEN REPLICA NO ACK message: ####%s\n", workerID, msg.Body)
				f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err == nil {
					f.WriteString(entry)
					f.Close()
				}
				panic("ERROR")
			}

			entry := fmt.Sprintf("[%s] [%s]   Received message: %s\n", now, workerID, msg.Body)

			f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err == nil {
				f.WriteString(entry)
				f.Close()
				d.Ack(false) // confirm this message only
			} else {
				log.Println("Failed to write to log file:", err)
			}
		}
	}()

	log.Println("Worker started. Waiting for messages.")
	<-forever
	log.Println("Worker finished. Closing.")
}
