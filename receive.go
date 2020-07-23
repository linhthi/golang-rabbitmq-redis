package main

import (
	"log"
	"os"
	"fmt"
	"net/smtp"
	"encoding/json"
	"github.com/go-redis/redis"
	"github.com/streadway/amqp"
)

type Message struct {
	Receiver string `json:"receiver"`
	Content string	`json:"content"`
	ID string `json:"id"`
}

type Job struct {
    ID string   `json:"id"`
    Status string   `json:"status"`
}

// smtpServer data to smtp server
type smtpServer struct {
	host string
	port string
}

// Address URI to smtp server
func (s *smtpServer) Address() string {
	return s.host + ":" + s.port
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
			saveInRedis(mess.ID, "running")
			sendMail(mess)
			saveInRedis(mess.ID, "completed")
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


func sendMail(message Message) {
	// Sender data.
    from := "hoanglinh831999@gmail.com"
    password := "hoanglinh1999"
    // Receiver email address.
    to := []string{message.Receiver}
    // smtp server configuration.
    smtpServer := smtpServer{host: "smtp.gmail.com", port: "587"}
    // Message.
    mess := []byte(message.Content)
    // Authentication.
    auth := smtp.PlainAuth("", from, password, smtpServer.host)
    // Sending email.
    err := smtp.SendMail(smtpServer.Address(), auth, from, to, mess)
    if err != nil {
        fmt.Println(err)
        return
    }
    fmt.Println("Email Sent!")

}


func saveInRedis(jobId string, status string) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		Password: "",
		DB: 0,
    })
	
	key := "jobId" + jobId

    err := client.Set(client.Context(), key, status , 0).Err()
    if err != nil {
        fmt.Println(err)
	}
	
    val, err := client.Get(client.Context(), key).Result()
    if err != nil {
        fmt.Println(err)
	}
	
    fmt.Println(val)
}



