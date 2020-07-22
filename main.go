package main

import(
	"fmt"
	"log"
	"net/http"
	"encoding/json"
	"os"
    "github.com/streadway/amqp"
    "github.com/go-redis/redis"
    "strconv"
)

type Message struct {
    JobId int
	Content string
    Sender string
    Status string
}

func main()  {
	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe("localhost:8000", nil))
}

func handler(w http.ResponseWriter, r *http.Request)  {
	var mess Message

	err := json.NewDecoder(r.Body).Decode(&mess)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

    mess.Status = checkCache("jobId" + strconv.Itoa(mess.JobId))
    if mess.Status == "completed" {
        fmt.Fprintf(w, "Message: %+v", mess)
        fmt.Println("Message: %v", mess)
        return
    } else {
        mess.Status = "running"
    }
    body, err := json.Marshal(mess)
    fmt.Fprintf(w, "Message: %+v", mess)
    fmt.Println("Message: %v", mess)

	// Push to Message Queue
	url := os.Getenv("AMQP_URL")
    if url == "" {
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
    err = channel.ExchangeDeclare(
        "events",
        "topic",
        true, 
        false, 
        false, 
        false, 
        nil,
    )
    if err != nil {
        panic(err)
    }

    message := amqp.Publishing{
        ContentType: "application/json",
        Body: body,
    }

    err = channel.Publish(
        "events", 
        "random-key", 
        false, 
        false, 
        message,
    )
    if err != nil {
        panic("error publishing a message to the queue:" + err.Error())
    }

    _, err = channel.QueueDeclare(
        "test", 
        true, 
        false, 
        false, 
        false, 
        nil,
    )
    if err != nil {
        panic("error declaring the queue: " + err.Error())
    }

    err = channel.QueueBind(
        "test", 
        "#", 
        "events", 
        false, 
        nil,
    )
    if err != nil {
        panic("error binding to the queue: " + err.Error())
    }
}


func checkCache(key string) string {
    client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		Password: "",
		DB: 0,
    })
	
    val, err := client.Get(client.Context(), key).Result()
    if err != nil {
        fmt.Println(err)
    }
    
    return val
}
