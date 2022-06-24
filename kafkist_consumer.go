package main

import (
	"encoding/json"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/gorilla/mux"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"
)

//StateStack is a stack of items
type StateStack struct {
	items []string
	lock  sync.RWMutex
}

//New is a function to create a new stack of item
func (s *StateStack) New() *StateStack {
	//1. simply create an empty slices of Item
	//2. stack is initialised with the empty slice
	s.items = []string{}
	return s
}

//Push is a function to add a new item into the stack
func (s *StateStack) Push(t string) {
	//3. lock before append to prevent different goroutine update it
	s.lock.Lock()
	s.items = append(s.items, t)
	//4. unlock before append to prevent different goroutine update it
	s.lock.Unlock()
}

//Pop is a function to get and remove a new item from the stack
//remember the LIFO behavior of the stack
func (s *StateStack) Pop() string {
	//5. lock before get to prevent different goroutine update it
	s.lock.Lock()
	//6. get the item with the greatest index
	item := s.items[len(s.items)-1]
	//7. make a new slice where the last index is excluded
	s.items = s.items[0 : len(s.items)-1]
	//8. unlock before append to prevent different goroutine update it
	s.lock.Unlock()
	//9. return
	return item
}

//Peek is to check the current state without removing it
//from the stack
//this is read operation so no locking is needed
func (s *StateStack) Peek() string {
	return s.items[len(s.items)-1]
}

type Consumer struct {
	ReplayMode bool   `yaml:"replay-mode"`
	ReplayType string `yaml:"replay-type"`
	ReplayFrom string `yaml:"replay-from"`
}
type Config struct {
	ServingPort int      `yaml:"serving_port"`
	KafkaBroker string   `yaml:"kafka_broker"`
	KafkaTopic  string   `yaml:"kafka_topic"`
	Hashtags    []string `yaml:"hashtags"`
	Consumer    Consumer `yaml:"consumer"`
}

const (
	START       = "start"
	STARTED     = "started"
	STOP        = "stop"
	STOPPED     = "stopped"
	RESUME      = "resume"
	PAUSE       = "pause"
	PAUSED      = "paused"
	SUCCESS_MSG = "Successfully %s from %s"
)

var (
	cfg    = Config{}
	stop   = false
	states = StateStack{}
)

//BrowserPayload is a payload for the rest api
type BrowserPayload struct {
	TopicName     string `json:"topic_name"`
	Broker        string `json:"broker"`
	Offset        string `json:"offset_start"`
	ConsumerGroup string `json:"group"`
}

func init() {
	states.New()
	states.Push(STOPPED)
	configLocation := os.Getenv("CONFIG_FILE")
	if yFile, err := ioutil.ReadFile(configLocation); err == nil {
		if errUnmarshall := yaml.Unmarshal(yFile, &cfg); errUnmarshall != nil {
			log.Fatalf("error while unmarshalling %s %v", configLocation, errUnmarshall)
		}
	} else {
		log.Fatalf("error while reading %s %v", configLocation, err)
	}
}
func isAllowed(action string) bool {
	switch states.Peek() {
	case STARTED:
		if action == STOP || action == PAUSE {
			return true
		}
		return false
	case PAUSED:
		if action == RESUME || action == STOP {
			return true
		}
		return false
	case STOPPED:
		if action == START {
			return true
		}
	default:
		return false
	}
	return false

}

//handleLifeCycle implement the change on the status request
//the status is part of the request path parameter
//PATH /topic/start
func handleLifeCycle(writer http.ResponseWriter, request *http.Request) {
	//stop = true
	//extract the path parameter from request
	reqUri := mux.Vars(request)

	if action, ok := reqUri["action"]; ok {
		//if action == START {
		//if it needs to start, check the current state
		if isAllowed(action) {
			handle(writer, request, action)
		} else {
			http.Error(writer, fmt.Sprintf("It is not allowed to %s on a %s state", action, states.Peek()), http.StatusBadRequest)
		}
	} else {
		if _, errW := writer.Write([]byte("Please provide action start, stop,resume,pause")); errW != nil {
			writer.WriteHeader(http.StatusBadRequest)
			http.Error(writer, errW.Error(), http.StatusInternalServerError)
		}
	}
}
func applyState(writer http.ResponseWriter, action string) {
	var previousState string
	switch states.Peek() {
	case STARTED:
		if action == STOP {
			//currentState = STOPPED
			previousState = states.Pop()
			states.Push(STOPPED)

		}
		if action == PAUSE {
			//currentState = PAUSED
			previousState = states.Pop()
			states.Push(PAUSED)

		}
		stop = true
	case PAUSED:
		if action == RESUME {
			//currentState = STARTED
			previousState = states.Pop()
			states.Push(STARTED)
			stop = false
		}
		if action == STOP {
			//currentState = STOPPED
			previousState = states.Pop()
			states.Push(STOPPED)
			stop = true
		}
	case STOPPED:
		if action == START {
			//currentState = STARTED
			previousState = states.Pop()
			states.Push(STARTED)
			stop = false
		}
	}
	if _, errWrite := writer.Write([]byte(fmt.Sprintf(SUCCESS_MSG, action, previousState))); errWrite != nil {
		http.Error(writer, errWrite.Error(), http.StatusInternalServerError)
	}
}
func handle(writer http.ResponseWriter, request *http.Request, action string) {
	switch action {
	case START:
		doStart(writer, request)

		break

	default:
		//just update the state accordingly
		applyState(writer, action)

	}
}
func doStart(writer http.ResponseWriter, request *http.Request) {
	p := BrowserPayload{}
	decoder := json.NewDecoder(request.Body)
	if err := decoder.Decode(&p); err != nil {
		log.Printf("error while encoding %v %v\n", request.Body, err)
		http.Error(writer, err.Error(), http.StatusInternalServerError)
	} else {

		if c, err := kafka.NewConsumer(&kafka.ConfigMap{
			"bootstrap.servers": p.Broker,
			"group.id":          p.ConsumerGroup,
			"auto.offset.reset": p.Offset,
		}); err != nil {
			panic(err)
		} else {
			if err := c.Subscribe(p.TopicName, nil); err != nil {
				if _, errWr := writer.Write([]byte(fmt.Sprintf("Error while subscribing to kafka topic %v ",
					err))); errWr != nil {
					http.Error(writer, errWr.Error(), http.StatusInternalServerError)
				}
			} else {

				go consume(c)
				writer.Write([]byte("starting..."))

			}

		}
	}
}
func transit(targetState string) {
	//state exchange
	//1. pop
	prevState := states.Pop()
	//2. push
	states.Push(targetState)
	log.Printf("Change from %s to %s\n", prevState, targetState)

}
func consume(consumer *kafka.Consumer) {
	var offset string
	var messageTimeStamp int64
	log.Printf("start reading\n")
	stop = false
	for {
		//log.Printf("current stop flag %v\n", stop)
		if !stop {
			if message, err := consumer.ReadMessage(-1); err == nil {
				offset = message.TopicPartition.Offset.String()
				messageTimeStamp = message.Timestamp.UnixMilli()
				log.Printf("%s timestamp %v diff current and data timestamp %v\n", offset, messageTimeStamp, time.Now().UnixMilli()-message.Timestamp.UnixMilli())
			} else {
				log.Printf("error %v", err)
			}
		} else {
			log.Printf("stop reading at offset %s for timestamp %v\n", offset, messageTimeStamp)
			break
		}

		//executed <- true
	}

}
func main() {
	log.Default().SetFlags(log.LstdFlags | log.Lshortfile)
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	route := mux.NewRouter()
	//route.HandleFunc("/read", handleLifeCycle).Methods("GET")
	route.HandleFunc("/topic/{action}", handleLifeCycle).Methods("PUT", "GET")
	s := fmt.Sprintf(":%d", cfg.ServingPort)
	go func() {
		log.Printf("starting server at port %d \n", cfg.ServingPort)
		if err := http.ListenAndServe(s, route); err != nil {
			log.Fatalf("Cannot start the server %v \n", err)
		}
	}()
	//handleReplay(tweetist.Daterange{TimestampStart: "2022-02-02T09:30:50Z"})

	<-ch
	log.Printf("Stopping the server...\n")

}
