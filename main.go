package main

import(
    "os"
	"fmt"
	"net/http"
    "encoding/json"
    "strconv"
    "github.com/streadway/amqp"
    "github.com/go-redis/redis"
    "github.com/gorilla/mux"
)

type Message struct {
    Receiver string `json:"receiver"`
    Content string  `json:"content"`
    ID string `json:"id"`
}

type Job struct {
    ID string   `json:"id"`
    Status string   `json:"status"`
}

var cnt int 

func main()  {
    cnt = 0
    router := mux.NewRouter()
    router.HandleFunc("/task", createTask).Methods("POST")
    router.HandleFunc("/tasks/{id}", getTask).Methods("GET")
	http.ListenAndServe(":8000", router)
}

func createTask(w http.ResponseWriter, r *http.Request)  {
	var mess Message

	err := json.NewDecoder(r.Body).Decode(&mess)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

    cnt++   
    mess.ID = strconv.Itoa(cnt)
    body, err := json.Marshal(mess)
    fmt.Fprintf(w, "Message: %+v", mess)
    fmt.Println("Message: %v", mess)
    saveInRedis(mess.ID, "in queue")

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

func getTask(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    params := mux.Vars(r)
    result := checkCache("jobId" + params["id"])
    if result == "" {
        json.NewEncoder(w).Encode("can't find this job")
    } else {
        job := Job{params["id"], result}
        json.NewEncoder(w).Encode(job)
    }
}

// Check status of job from redis
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



