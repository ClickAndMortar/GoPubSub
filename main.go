package main

import (
	"context"
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v2"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"

	"cloud.google.com/go/pubsub"
)

var (
	client *pubsub.Client

	// Messages received by this instance.
	messagesMu sync.Mutex
	messages   map[string][]string

	// token is used to verify push requests.
	token = "abcd123"

	topicName        string
	subscriptionName string

	page Page

	topics        map[string]*pubsub.Topic
	subscriptions map[string]*pubsub.Subscription
)

type Config struct {
	Project string

	Topics []struct {
		Name         string
		Subscription string
		Payloads     []struct {
			Name    string
			Payload string
		}
	}
}

type Page struct {
	Config   Config
	Messages map[string][]string
}

func main() {
	page = Page{}
	config := Config{}

	configData, err := ioutil.ReadFile("config.yaml")
	configErr := yaml.Unmarshal([]byte(configData), &config)
	if configErr != nil {
		log.Fatal(configErr)
	}

	page.Config = config

	os.Setenv("PUBSUB_VERIFICATION_TOKEN", "abcd123")

	ctx := context.Background()

	client, err = pubsub.NewClient(ctx, config.Project)
	if err != nil {
		log.Fatal(err)
	}

	messages = make(map[string][]string)
	page.Messages = messages

	topics = make(map[string]*pubsub.Topic)
	subscriptions = make(map[string]*pubsub.Subscription)

	// Create the topic if it doesn't exist.
	for _, topicConfig := range config.Topics {
		topicName = topicConfig.Name
		topics[topicName] = client.Topic(topicName)

		exists, err := topics[topicName].Exists(ctx)
		if err != nil {
			log.Fatal(err)
		}
		if !exists {
			log.Printf("Topic %v doesn't exist - creating it", topicName)
			_, err = client.CreateTopic(ctx, topicName)
			if err != nil {
				log.Fatal(err)
			}
		}

		subscriptionName = topicConfig.Subscription
		if subscriptionName == "" {
			subscriptionName = fmt.Sprintf("sub-%s", topicName)
			log.Printf("No subscription name given for topic %s, using %s", topicName, subscriptionName)
		}
		subscriptions[subscriptionName] = client.Subscription(subscriptionName)
		subExists, err := subscriptions[subscriptionName].Exists(ctx)
		if err != nil {
			log.Fatal(err)
		}
		if !subExists {
			log.Printf("Subscription %v doesn't exist - creating it", subscriptionName)
			subConfig := pubsub.SubscriptionConfig{
				Topic: topics[topicName],
			}
			_, err = client.CreateSubscription(ctx, subscriptionName, subConfig)
			if err != nil {
				log.Fatal(err)
			}
		}

		go pullMessages(ctx, subscriptions[subscriptionName], topics[topicName])
	}

	http.HandleFunc("/", listHandler)
	http.HandleFunc("/publish", publishHandler)
	http.HandleFunc("/pubsub/push", pushHandler)

	port := os.Getenv("GOPUBSUB_PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}

	log.Printf("Listening on http://127.0.0.1:%s", port)
	http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
}

func pullMessages(ctx context.Context, subscription *pubsub.Subscription, topic *pubsub.Topic) {
	received := 0
	cctx, cancel := context.WithCancel(ctx)
	err := subscription.Receive(cctx, func(ctx context.Context, msg *pubsub.Message) {
		msg.Ack()
		fmt.Printf("Got message: %q\n", string(msg.Data))
		messages[topic.ID()] = append(messages[topic.ID()], string(msg.Data))
		messagesMu.Lock()
		defer messagesMu.Unlock()
		received++
		if received == 10 {
			cancel()
		}
	})
	if err != nil {
		fmt.Printf("Receive: %v", err)
	}
}

type pushRequest struct {
	Message struct {
		Attributes map[string]string
		Data       []byte
		ID         string `json:"message_id"`
	}
	Subscription string
}

func pushHandler(w http.ResponseWriter, r *http.Request) {
	// Verify the token.
	if r.URL.Query().Get("token") != token {
		http.Error(w, "Bad token", http.StatusBadRequest)
	}
	msg := &pushRequest{}
	if err := json.NewDecoder(r.Body).Decode(msg); err != nil {
		http.Error(w, fmt.Sprintf("Could not decode body: %v", err), http.StatusBadRequest)
		return
	}

	messagesMu.Lock()
	defer messagesMu.Unlock()
	// Limit to ten.
	/*messages = append(messages, string(msg.Message.Data))
	if len(messages) > maxMessages {
		messages = messages[len(messages)-maxMessages:]
	}*/
}

func listHandler(w http.ResponseWriter, r *http.Request) {
	messagesMu.Lock()
	defer messagesMu.Unlock()

	if err := tmpl.Execute(w, page); err != nil {
		log.Printf("Could not execute template: %v", err)
	}
}

func publishHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	msg := &pubsub.Message{
		Data: []byte(r.FormValue("payload")),
	}

	publishTopic := client.Topic(r.FormValue("topic"))
	if _, err := publishTopic.Publish(ctx, msg).Get(ctx); err != nil {
		http.Error(w, fmt.Sprintf("Could not publish message: %v", err), 500)
		return
	}

	fmt.Fprint(w, "Message published.")
}

var tmpl = template.Must(template.ParseFiles("template.html"))
