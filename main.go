package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/alexandrevicenzi/go-sse"
	"gopkg.in/yaml.v2"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"cloud.google.com/go/pubsub"
)

var (
	client *pubsub.Client

	messagesMu sync.Mutex
	messages   map[string][]pubsub.Message

	topicName        string
	subscriptionName string

	page Page

	topics        map[string]*pubsub.Topic
	subscriptions map[string]*pubsub.Subscription

	maxMessages int

	sseServer *sse.Server
	sseJson   []byte
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
	Messages map[string][]pubsub.Message
}

type PublishResponse struct {
	ID    string `json:"id"`
	Topic string `json:"topic"`
}

func main() {
	page = Page{}
	config := Config{}

	sseServer = sse.NewServer(nil)
	defer sseServer.Shutdown()

	maxMessages, _ = strconv.Atoi(getEnvDefault("GOPUBSUB_MAX_MESSAGES", "10"))

	configPath := getEnvDefault("GOPUBSUB_CONFIG", "config.yaml")
	configData, err := ioutil.ReadFile(configPath)
	configErr := yaml.Unmarshal([]byte(configData), &config)
	if configErr != nil {
		log.Fatal(configErr)
	}

	page.Config = config

	ctx := context.Background()

	client, err = pubsub.NewClient(ctx, config.Project)
	if err != nil {
		log.Fatal(err)
	}

	messages = make(map[string][]pubsub.Message)
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
	http.HandleFunc("/messages", messagesHandler)
	http.Handle("/events/", sseServer)

	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	port := getEnvDefault("GOPUBSUB_PORT", "8080")

	log.Printf("Listening on http://127.0.0.1:%s", port)
	http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
}

func getEnvDefault(name string, defaultValue string) string {
	variable := os.Getenv(name)
	if variable == "" && defaultValue != "" {
		log.Printf("Environment variable %s not set or empty, using default %s", name, defaultValue)
		variable = defaultValue
	}

	return variable
}

func pullMessages(ctx context.Context, subscription *pubsub.Subscription, topic *pubsub.Topic) {
	cctx, _ := context.WithCancel(ctx)
	err := subscription.Receive(cctx, func(ctx context.Context, msg *pubsub.Message) {
		msg.Ack()
		log.Printf("Topic [%s], subscription [%s], got message: %q\n", topic.ID(), subscription.ID(), string(msg.Data))
		messages[topic.ID()] = append([]pubsub.Message{*msg}, messages[topic.ID()]...)

		messagesMu.Lock()
		defer messagesMu.Unlock()
		if len(messages[topic.ID()]) > maxMessages {
			messages[topic.ID()] = messages[topic.ID()][:maxMessages]
		}

		sseJson, _ = json.Marshal(messages)
		sseServer.SendMessage("/events/messages", sse.SimpleMessage(string(sseJson)))
	})
	if err != nil {
		fmt.Printf("Receive: %v", err)
	}
}

func listHandler(w http.ResponseWriter, r *http.Request) {
	messagesMu.Lock()
	defer messagesMu.Unlock()

	if err := tmpl.Execute(w, page); err != nil {
		log.Printf("Could not execute template: %v", err)
	}
}

func messagesHandler(w http.ResponseWriter, r *http.Request) {
	if strings.ContainsAny(r.Header.Get("Accept"), "application/json") {
		w.Header().Set("Content-Type", "application/json")
		jsonResponse, _ := json.Marshal(messages)
		w.Write(jsonResponse)
	}
}

func publishHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	msg := &pubsub.Message{
		Data: []byte(strings.TrimSpace(r.FormValue("payload"))),
	}

	publishTopic := client.Topic(r.FormValue("topic"))
	serverId, err := publishTopic.Publish(ctx, msg).Get(ctx)
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not publish message: %v", err), 500)
		return
	}

	response := PublishResponse{
		ID:    serverId,
		Topic: publishTopic.ID(),
	}

	jsonResponse, _ := json.Marshal(response)
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResponse)
}

var tmpl = template.Must(template.ParseFiles("template.html"))
