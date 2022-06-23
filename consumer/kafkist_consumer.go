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
	START   = "start"
	STARTED = "started"
	STOP    = "stop"
	STOPPED = "stopped"
	RESUME  = "resume"
	PAUSE   = "pause"
	PAUSED  = "paused"
)

var (
	cfg          = Config{}
	stop         = false
	currentState = STOPPED
)

//BrowserPayload is a payload for the rest api
type BrowserPayload struct {
	TopicName      string `json:"topic_name"`
	Broker         string `json:"broker"`
	TimestampBegin string `json:"timestamp_begin"`
	TimestampEnd   string `json:"timestamp_end"`
	Offset         string `json:"offset_start"`
	ConsumerGroup  string `json:"group"`
}

func init() {
	currentState = STOPPED
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
	switch currentState {
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
			http.Error(writer, fmt.Sprintf("It is not allowed to %s on a %s state", action, currentState), http.StatusBadRequest)
		}
	} else {
		if _, errW := writer.Write([]byte("Please provide action start, stop,resume,pause")); errW != nil {
			writer.WriteHeader(http.StatusBadRequest)
			http.Error(writer, errW.Error(), http.StatusInternalServerError)
		}
	}
}
func getNextState(writer http.ResponseWriter, action string) {
	switch currentState {
	case STARTED:
		if action == STOP {
			currentState = STOPPED

		}
		if action == PAUSE {
			currentState = PAUSED

		}
		stop = true
	case PAUSED:
		if action == RESUME {
			currentState = STARTED
			stop = false
		}
		if action == STOP {
			currentState = STOPPED
			stop = true
		}
	case STOPPED:
		if action == START {
			currentState = STARTED
			stop = false
		}
	}
	if _, errWrite := writer.Write([]byte(fmt.Sprintf("Successfully %s from %s", action, currentState))); errWrite != nil {
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
		getNextState(writer, action)

	}
}
func doStart(writer http.ResponseWriter, request *http.Request) {
	p := BrowserPayload{}
	decoder := json.NewDecoder(request.Body)
	if err := decoder.Decode(&p); err != nil {
		log.Printf("error while encoding %v %v\n", request.Body, err)
		http.Error(writer, err.Error(), http.StatusInternalServerError)
	} else {

		if c, err := kafka.NewConsumer(&kafka.ConfigMap{"metadata.broker.list": p.Broker,
			"security.protocol":               "PLAINTEXT",
			"group.id":                        p.ConsumerGroup,
			"auto.offset.reset":               p.Offset,
			"go.application.rebalance.enable": true,
			"request.timeout.ms":              60000,
			"go.events.channel.enable":        true}); err != nil {
			panic(err)
		} else {
			if err := c.Subscribe(p.TopicName, nil); err != nil {
				if _, errWr := writer.Write([]byte(fmt.Sprintf("Error while subscribing to kafka topic %v ",
					err))); errWr != nil {
					http.Error(writer, errWr.Error(), http.StatusInternalServerError)
				}
			} else {
				//use go routine
				//isExecuted := make(chan bool)

				go consume(c)
				//<-isExecuted
				writer.Write([]byte("starting..."))

			}

		}
	}
}
func consume(consumer *kafka.Consumer) {
	var offset string
	var messageTimeStamp int64
	log.Printf("start reading\n")
	stop = false
	currentState = STARTED
	for {
		//log.Printf("current stop flag %v\n", stop)
		if !stop {
			if message, err := consumer.ReadMessage(time.Second * 10); err == nil {
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
	route.HandleFunc("/topic/{action}", handleLifeCycle).Methods("PUT")
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
