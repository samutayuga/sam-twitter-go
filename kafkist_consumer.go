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
	"text/template"
	"time"
)

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
	START      = "start"
	STARTED    = "started"
	STOP       = "stop"
	STOPPED    = "stopped"
	RESUME     = "resume"
	PAUSE      = "pause"
	PAUSED     = "paused"
	SuccessMsg = "Successfully %s from %s"
)

var (
	cfg             = Config{}
	stop            = false
	states          = StateStack{}
	currentInstance = BrowserPayload{}
	msgTemplate     *template.Template
)

//var currentInstance BrowserPayload

//BrowserPayload is a payload for the rest api
type BrowserPayload struct {
	TopicName     string `json:"topic_name,omitempty"`
	Broker        string `json:"broker,omitempty"`
	Offset        string `json:"offset_start,omitempty"`
	ConsumerGroup string `json:"group,omitempty"`
	State         string `json:"state,omitempty"`
}

func init() {
	//initialize the template
	var err error
	if msgTemplate, err = template.ParseGlob("*.gotmpl"); err != nil {
		log.Fatalf("error while parsing the template %v\n", err)
	} else {
		log.Printf("template %s to display is initialized\n", msgTemplate.Name())
	}
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
	methodName := request.Method

	if action, ok := reqUri["action"]; ok {
		log.Printf("%s request with path param %s\n", methodName, action)
		//if action == START {
		//if it needs to start, check the current state
		if isAllowed(action) {
			handle(writer, request, action)
		} else {
			msg := fmt.Sprintf("It is not allowed to %s on a %s state", action, states.Peek())
			log.Println(msg)
			http.Error(writer, msg, http.StatusBadRequest)
		}
	} else {
		if _, errW := writer.Write([]byte("Please provide action start, stop,resume,pause")); errW != nil {
			writer.WriteHeader(http.StatusBadRequest)
			http.Error(writer, errW.Error(), http.StatusInternalServerError)
		}
	}
}
func getState(writer http.ResponseWriter, request *http.Request) {
	log.Printf("%s request\n", request.Method)
	currentInstance.State = states.Peek()
	if content, err := json.Marshal(currentInstance); err == nil {
		if _, errW := writer.Write(content); errW != nil {
			http.Error(writer, errW.Error(), http.StatusInternalServerError)
		}
	}

}
func applyState(writer http.ResponseWriter, action string) {
	prevState := states.Peek()
	switch prevState {
	case STARTED:
		if action == STOP {
			transit(STOPPED)

		}
		if action == PAUSE {
			transit(PAUSED)

		}
		stop = true
	case PAUSED:
		if action == RESUME {
			transit(STARTED)
			stop = false
		}
		if action == STOP {
			transit(STOPPED)
			stop = true
		}
	case STOPPED:
		if action == START {
			transit(STARTED)
			stop = false
		}
	}
	if _, errWrite := writer.Write([]byte(fmt.Sprintf(SuccessMsg, action, prevState))); errWrite != nil {
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
func ping(writer http.ResponseWriter, request *http.Request) {
	writer.WriteHeader(http.StatusOK)
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
				transit(STARTED)
				currentInstance = p

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
	queueTime := DurationQueue{}
	isDone := make(chan bool)
	go ConsumeQueue(queueTime, isDone)
	for {
		//log.Printf("current stop flag %v\n", stop)
		if !stop {
			if message, err := consumer.ReadMessage(-1); err == nil {
				offset = message.TopicPartition.Offset.String()
				//messageTimeStamp = message.Timestamp.UnixMilli()
				timeDuration := time.Since(message.Timestamp)
				queueTime.Enqueue(timeDuration)

				txt := fmt.Sprintf("current time %v offset %s was posted %s ago", time.Now().Format("2006-01-02 15:04:05.000"), offset, timeDuration.Truncate(time.Millisecond).String())
				if errDisplay := msgTemplate.ExecuteTemplate(os.Stdout, "displayer.gotmpl", txt); errDisplay != nil {
					log.Printf("error while writing into template %v\n", errDisplay)
				}
			} else {
				log.Printf("error %v", err)
			}
		} else {
			if states.Peek() == PAUSED {
				log.Printf("get signal to pause...\n")
				time.Sleep(time.Second * 5)
			}
			if states.Peek() == STOPPED {
				log.Printf("stop reading at offset %s for timestamp %v\n", offset, messageTimeStamp)
				consumer.Unsubscribe()
				log.Printf("unsubcribe to topic....\n")
				consumer.Close()
				log.Printf("close the consumer....\n")
				break
			}

		}

		//executed <- true
	}
	log.Printf("bye....\n")

}
func main() {
	log.Default().SetFlags(log.LstdFlags | log.Lshortfile)
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	route := mux.NewRouter()
	//route.HandleFunc("/read", handleLifeCycle).Methods("GET")
	route.HandleFunc("/", ping).Methods(http.MethodGet)
	route.HandleFunc("/topic", getState).Methods(http.MethodGet)
	//route.HandleFunc("/topic/start", doStart).Methods(http.MethodPost)
	route.HandleFunc("/topic/{action}", handleLifeCycle).Methods(http.MethodPost, http.MethodPatch)
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
