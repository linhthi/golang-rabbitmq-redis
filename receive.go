package main

import (
	"log"
	"os"
	"github.com/streadway/amqp"
	"strconv"
	"fmt"
	"encoding/json"
	"github.com/go-redis/redis"
)

type Message struct {
	JobId int
	Content string
    Sender string
    Status string
}

func main() {
	url := os.Getenv("AMQP_URL")
    if url == "" {
        url = "amqp://guest:guest@localhost:5672"
    }

    // Connect to the rabbitMQ instance
    conn, err := amqp.Dial(url)
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"test", // name
		true,   // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		nil,     // arguments
	)
	failOnError(err, "Failed to declare a queue")

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	failOnError(err, "Failed to register a consumer")
	
	forever := make(chan bool)
	

	// Send message
	go func() {
		for d := range msgs {
			log.Printf("Received a message: %s", d.Body)
			var mess Message
			json.Unmarshal(d.Body, &mess)

			saveInRedis(mess.JobId, mess.Status)
		}
	}()
	
	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}

func failOnError(err error, msg string) {
	if err != nil {
	  log.Fatalf("%s: %s", msg, err)
	}
}


func sendMail() {

}


func saveInRedis(jobId int, status string) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		Password: "",
		DB: 0,
    })
	
	key := "jobId" + strconv.Itoa(jobId)

    err := client.Set(client.Context(), key, "completed" , 0).Err()
    if err != nil {
        fmt.Println(err)
	}
	
    val, err := client.Get(client.Context(), key).Result()
    if err != nil {
        fmt.Println(err)
	}
	
    fmt.Println(val)
}



