package main

import (
	"encoding/json"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/gorilla/mux"
	"github.com/samutayuga/sam-twitter-go/tweetist"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

var (
	cfg = tweetist.Config{}
)

//BrowserPayload is a payload for the rest api
type BrowserPayload struct {
	TopicName      string `json:"topic_name"`
	Broker         string `json:"broker"`
	TimestampBegin string `json:"timestamp_begin"`
	TimestampEnd   string `json:"timestamp_end"`
}

func init() {
	configLocation := os.Getenv("CONFIG_FILE")
	if yFile, err := ioutil.ReadFile(configLocation); err == nil {
		if errUnmarshall := yaml.Unmarshal(yFile, &cfg); errUnmarshall != nil {
			log.Fatalf("error while unmarshalling %s %v", configLocation, errUnmarshall)
		}
	} else {
		log.Fatalf("error while reading %s %v", configLocation, err)
	}
}
func getPartitionNumbers(pars []kafka.TopicPartition) string {
	var pNums string
	for i, par := range pars {
		if i == len(pars)-1 {
			pNums = pNums + strconv.Itoa(int(par.Partition))
		} else {
			pNums = pNums + strconv.Itoa(int(par.Partition)) + ", "
		}
	}

	return pNums
}
func resetPatititionOffsetToTimestamps(c *kafka.Consumer, partitions []kafka.TopicPartition, timestamp int64) ([]kafka.TopicPartition, error) {
	var prs []kafka.TopicPartition
	for _, par := range partitions {
		offset := kafka.Offset(timestamp)
		tp := kafka.TopicPartition{Topic: par.Topic, Partition: par.Partition, Offset: offset}
		log.Printf("The starting offset is %v\n", tp.Offset)

		prs = append(prs, tp)
	}
	if updtPars, err := c.OffsetsForTimes(prs, 5000); err != nil {
		log.Printf("Failed to reset offsets to supplied timestamp due to error: %v\n", err)
		return updtPars, err
	} else {
		return updtPars, nil

	}

}
func handleStart(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "application/json;charset=UFT-8")
	p := tweetist.Daterange{}
	decoder := json.NewDecoder(request.Body)
	if err := decoder.Decode(&p); err != nil {
		log.Fatalf("error while encoding %v %v\n", request.Body, err)
	} else {
		resp := make(chan bool)
		go handleReplay(resp, p)
		<-resp
		writer.WriteHeader(http.StatusOK)
	}

}
func handlePause(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "application/json;charset=UFT-8")
	p := tweetist.Daterange{}
	decoder := json.NewDecoder(request.Body)
	if err := decoder.Decode(&p); err != nil {
		log.Fatalf("error while encoding %v %v\n", request.Body, err)
	} else {
		resp := make(chan bool)
		go handleReplay(resp, p)
		<-resp
		writer.WriteHeader(http.StatusOK)
	}
}
func handleBrowse(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "application/json;charset=UFT-8")
	p := BrowserPayload{}
	decoder := json.NewDecoder(request.Body)
	if err := decoder.Decode(&p); err != nil {
		log.Fatalf("error while encoding %v %v\n", request.Body, err)
	} else {

		if c, err := kafka.NewConsumer(&kafka.ConfigMap{"metadata.broker.list": p.Broker,
			"security.protocol":               "PLAINTEXT",
			"group.id":                        "sam",
			"auto.offset.reset":               "earliest",
			"go.application.rebalance.enable": true,
			"go.events.channel.enable":        true}); err != nil {
			panic(err)
		} else {
			c.Subscribe(p.TopicName, nil)
			consumeTopic(c, p)
		}
		writer.WriteHeader(http.StatusOK)
	}

}
func consumeTopic(c *kafka.Consumer, payload BrowserPayload) {
	var run = true
	for run == true {
		select {

		case ev := <-c.Events():
			switch e := ev.(type) {
			case kafka.AssignedPartitions:
				partAssign := e.Partitions
				if len(partAssign) == 0 {
					log.Println("No partitions assigned")
					continue
				}

				log.Printf("Assigned/Re-assigned Partitions: %s\n", getPartitionNumbers(partAssign))

				log.Printf("reset the offset to timestamp %v\n", payload.TimestampBegin)
				if t, err := time.Parse(time.RFC3339Nano, payload.TimestampBegin); err != nil {
					log.Fatalf("failed to parse replay timestamp %s due to error %v", payload.TimestampBegin, err)
				} else {
					if partToAssign, errAssign := resetPatititionOffsetToTimestamps(c, e.Partitions, t.UnixNano()/int64(time.Millisecond)); errAssign != nil {
						log.Fatalf("error trying to reset offsets to timestamp %v", errAssign)
					} else {
						c.Assign(partToAssign)
					}
				}
			case kafka.RevokedPartitions:
				c.Unassign()
			case *kafka.Message:
				fmt.Printf("%% Message on %v, offset %v, time stamp %v\n", *e.TopicPartition.Topic, e.TopicPartition.Offset, e.Timestamp)
			case kafka.PartitionEOF:
				fmt.Printf("%% Reached %v\n", e)
			case kafka.Error:
				fmt.Fprintf(os.Stderr, "%% Error: %v\n", e)
			}

		}

	}
}
func handleResume(writer http.ResponseWriter, request *http.Request) {

}
func handleReplay(resp chan bool, requestPayload tweetist.Daterange) {
	if c, err := kafka.NewConsumer(&kafka.ConfigMap{"metadata.broker.list": cfg.KafkaBroker,
		"security.protocol":               "PLAINTEXT",
		"group.id":                        "sam",
		"auto.offset.reset":               "earliest",
		"go.application.rebalance.enable": true,
		"go.events.channel.enable":        true}); err != nil {
		panic(err)
	} else {
		sigchan := make(chan os.Signal, 1)
		signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)
		c.Subscribe(cfg.KafkaTopic, nil)
		run := true
		for run == true {
			select {
			case isRunning := <-resp:
				log.Printf("Receive signal to %v\n", isRunning)
			case sig := <-sigchan:
				log.Printf("Caught signal %v: terminating\n", sig)
				run = false
			case ev := <-c.Events():
				switch e := ev.(type) {
				case kafka.AssignedPartitions:
					partAssign := e.Partitions
					if len(partAssign) == 0 {
						log.Println("No partitions assigned")
						continue
					}

					log.Printf("Assigned/Re-assigned Partitions: %s\n", getPartitionNumbers(partAssign))
					if cfg.Consumer.ReplayMode {
						switch cfg.Consumer.ReplayType {
						case "timestamp":
							resp <- true
							log.Printf("reset the offset to timestamp %v\n", requestPayload.TimestampStart)
							if t, err := time.Parse(time.RFC3339Nano, requestPayload.TimestampStart); err != nil {
								log.Fatalf("failed to parse replay timestamp %s due to error %v", requestPayload.TimestampStart, err)
							} else {
								if partToAssign, errAssign := resetPatititionOffsetToTimestamps(c, e.Partitions, t.UnixNano()/int64(time.Millisecond)); errAssign != nil {
									log.Fatalf("error trying to reset offsets to timestamp %v", errAssign)
								} else {
									c.Assign(partToAssign)
								}
							}

						case "frombeginning":

						}
					}
				case kafka.RevokedPartitions:
					c.Unassign()
				case *kafka.Message:
					fmt.Printf("%% Message on %v, offset %v, time stamp %v\n", *e.TopicPartition.Topic, e.TopicPartition.Offset, e.Timestamp)
				case kafka.PartitionEOF:
					fmt.Printf("%% Reached %v\n", e)
				case kafka.Error:
					fmt.Fprintf(os.Stderr, "%% Error: %v\n", e)
				}

			}

		}
	}
}
func main() {
	log.Default().SetFlags(log.LstdFlags | log.Lshortfile)
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	route := mux.NewRouter()
	route.HandleFunc("/start", handleStart).Methods("POST")
	route.HandleFunc("/pause", handlePause).Methods("POST")
	route.HandleFunc("/resume", handleResume).Methods("POST")
	route.HandleFunc("/browse", handleBrowse).Methods("GET")
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
