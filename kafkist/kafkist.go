package kafkist

import (
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"log"
	"net"
	"os"
)

var (
	producer *kafka.Producer
)

func getIpAddress() string {
	name, err := os.Hostname()
	if err != nil {
		fmt.Printf("Oops: %v\n", err)

	}

	addrs, err := net.LookupHost(name)
	if err != nil {
		fmt.Printf("Oops: %v\n", err)

	}

	for _, a := range addrs {
		return a
	}
	return ""
}
func CreateProducer(boostrap string) {
	var err error
	if producer, err = kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": boostrap,
		"client.id": getIpAddress(),
		"acks":      "all"}); err == nil {
		log.Printf("Kafka producer is instantiated %s \n", boostrap)
		log.Printf("Kafka producer %v \n", producer)

	} else {
		log.Fatalf("error while creating producer for %s %v", boostrap, err)
	}

}
func HandleReport() {
	for e := range producer.Events() {
		switch ev := e.(type) {
		case *kafka.Message:
			if ev.TopicPartition.Error != nil {
				log.Printf("error while delivering message %v\n",
					ev.TopicPartition)
			} else {
				log.Printf("message written into topic %v,partition %v, offset %v \n", *ev.TopicPartition.Topic, ev.TopicPartition.Partition, ev.TopicPartition.Offset)
			}

		}
	}
}
func Produce(topicName *string, message string) {
	//this is to handle the kafka delivery result
	//delChan := make(chan kafka.Event, 10000)
	if errProd := producer.Produce(&kafka.Message{TopicPartition: kafka.TopicPartition{Topic: topicName,
		Partition: kafka.PartitionAny}, Value: []byte(message)}, nil); errProd == nil {

		//channelOut := <-delChan
		//messageReport := channelOut.(*kafka.Message)
		//if messageReport.TopicPartition.Error != nil {
		//	log.Printf("error while delivering message %v\n",
		//		messageReport.TopicPartition.Error)
		//} else {
		//	log.Printf("message written into partition %d, offset %v, ts type %v, timestamp %v \n", messageReport.TopicPartition.Partition, messageReport.TopicPartition.Offset, messageReport.TimestampType, messageReport.Timestamp)
		//}
	} else {
		log.Printf("Error while pushing message to %v\n", errProd)

	}

}
