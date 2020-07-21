package main

import(
	"fmt"
	"log"
	"net/http"
	"encoding/json"
	"os"
	"github.com/streadway/amqp"
)

type Message struct {
	Content string
	Sender string
}

func main()  {
	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe("localhost:8000", nil))
}

func handler(w http.ResponseWriter, r *http.Request)  {
	var m Message

	err := json.NewDecoder(r.Body).Decode(&m)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fmt.Fprintf(w, "Message: %+v", m)


	// Push to Message Queue
	url := os.Getenv("AMQP_URL")

    //If it doesn't exist, use the default connection string.

    if url == "" {
        //Don't do this in production, this is for testing purposes only.
        url = "amqp://guest:guest@localhost:5672"
    }

    // Connect to the rabbitMQ instance
    connection, err := amqp.Dial(url)

    if err != nil {
        panic("could not establish connection with RabbitMQ:" + err.Error())
    }


    // Create a channel from the connection. We'll use channels to access the data in the queue rather than the
    // connection itself
    channel, err := connection.Channel()
    if err != nil {
        panic("could not open RabbitMQ channel:" + err.Error())
    }

    // create an exahange that will bind to the queue to send and receive messages
    err = channel.ExchangeDeclare("events", "topic", true, false, false, false, nil)
    if err != nil {
        panic(err)
    }

    message := amqp.Publishing{
        Body: []byte(m.Content),
    }

    err = channel.Publish("events", "random-key", false, false, message)
    if err != nil {
        panic("error publishing a message to the queue:" + err.Error())
    }

    _, err = channel.QueueDeclare("test", true, false, false, false, nil)
    if err != nil {
        panic("error declaring the queue: " + err.Error())
    }

    err = channel.QueueBind("test", "#", "events", false, nil)
    if err != nil {
        panic("error binding to the queue: " + err.Error())
    }




}