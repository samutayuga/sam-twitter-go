package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/gorilla/mux"
	"github.com/samutayuga/sam-twitter-go/kafkist"
	"github.com/samutayuga/sam-twitter-go/tweetist"
	"gopkg.in/yaml.v3"
)

var (
	cfg = tweetist.Config{}
)

func init() {
	configLocation := os.Getenv("CONFIG_FILE")
	if yFile, err := ioutil.ReadFile(configLocation); err == nil {
		if errUnmarshall := yaml.Unmarshal(yFile, &cfg); errUnmarshall != nil {
			log.Fatalf("error while unmarshalling %s %v", configLocation, errUnmarshall)
		}
	} else {
		log.Fatalf("error while reading %s %v", configLocation, err)
	}
	kafkist.CreateProducer(cfg.KafkaBroker)
	tweetist.TopicName = &cfg.KafkaTopic
	tweetist.Hashtags = cfg.Hashtags

}

func RequestHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		w.WriteHeader(http.StatusOK)
	default:
		log.Printf("not supported %s %s ", r.Method, r.RequestURI)
		w.WriteHeader(http.StatusBadRequest)
	}

}
func main() {
	log.Default().SetFlags(log.LstdFlags | log.Lshortfile)
	//launching main logic
	//1. call twittist package
	routes := mux.NewRouter()
	routes.HandleFunc("/", RequestHandler)
	s := fmt.Sprintf(":%d", cfg.ServingPort)
	go func() {
		log.Printf("starting server at port %d \n", cfg.ServingPort)
		if err := http.ListenAndServe(s, routes); err != nil {
			log.Fatalf("Cannot start the server %v \n", err)
		}
	}()
	//kafkist.CreateProducer()
	tweetist.DoRandomData()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	<-ch
	log.Printf("Stopping the server...\n")
}
