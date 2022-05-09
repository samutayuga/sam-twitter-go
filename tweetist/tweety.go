package tweetist

import (
	"flag"
	"fmt"
	"github.com/coreos/pkg/flagutil"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/samutayuga/sam-twitter-go/kafkist"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)
import "github.com/dghubble/oauth1"

type Daterange struct {
	TimestampStart string `json:"timestamp_start"`
	TimestampEnd   string `json:"timestamp_end"`
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

var (
	httpClient *http.Client
	client     *twitter.Client

	TopicName *string
	Hashtags  []string
)

func init() {
	consumerKeyFromEnv := os.Getenv("TWITTER_CONSUMER_KEY")
	consumerSecretFromEnv := os.Getenv("TWITTER_CONSUMER_SECRET")
	accessTokenFromEnv := os.Getenv("TWITTER_ACCESS_TOKEN")
	accessSecretFromEnv := os.Getenv("TWITTER_ACCESS_SECRET")
	flags := flag.NewFlagSet("user-auth", flag.ExitOnError)
	consumerKey := flags.String("consumer-key", consumerKeyFromEnv, "Twitter Consumer Key")
	consumerSecret := flags.String("consumer-secret", consumerSecretFromEnv, "Twitter Consumer Secret")
	accessToken := flags.String("access-token", accessTokenFromEnv, "Twitter Access Token")
	accessSecret := flags.String("access-secret", accessSecretFromEnv, "Twitter access secret")
	flags.Parse(os.Args[1:])
	flagutil.SetFlagsFromEnv(flags, "TWITTER")
	if *consumerKey == "" || *consumerSecret == "" || *accessToken == "" || *accessSecret == "" {
		log.Fatal("Consumer key/secret and Access token/secret required")
	}
	config := oauth1.NewConfig(*consumerKey, *consumerSecret)
	token := oauth1.NewToken(*accessToken, *accessSecret)
	httpClient = config.Client(oauth1.NoContext, token)
	client = twitter.NewClient(httpClient)

}

func DoStream() {
	fmt.Printf("Starting Stream for %v hashtags\n", Hashtags)
	filterParams := &twitter.StreamFilterParams{Track: Hashtags,
		StallWarnings: twitter.Bool(true)}

	if stream, err := client.Streams.Filter(filterParams); err != nil {
		log.Fatal(err)
	} else {
		//demux
		demux := twitter.NewSwitchDemux()
		demux.Tweet = func(tweet *twitter.Tweet) {
			text := tweet.Text
			log.Printf("pumping %s \n", text)
			kafkist.Produce(TopicName, text)
			go kafkist.HandleReport()
		}
		go demux.HandleChan(stream.Messages)
		ch := make(chan os.Signal)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		tweet := <-ch

		//push tweet into kafka
		log.Println(tweet)
		fmt.Println("Stopping Stream")
		stream.Stop()
	}
}
