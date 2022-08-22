package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/alexandrevicenzi/go-sse"
	"github.com/gobuffalo/packr"
	"gopkg.in/yaml.v2"

	"cloud.google.com/go/pubsub"
)

var (
	clients map[string]*pubsub.Client
	client  *pubsub.Client

	messagesMu sync.Mutex
	messages   map[string][]pubsub.Message

	topicName        string
	topicProject     string
	subscriptionName string

	page Page

	topics        map[string]*pubsub.Topic
	subscriptions map[string]*pubsub.Subscription

	maxMessages int

	sseServer *sse.Server
	sseJSON   []byte

	box packr.Box

	tmpl *template.Template
)

// The Config struct holds application configuration
type Config struct {
	Topics []struct {
		ID           string
		Name         string
		Project      string
		Subscription string
		Payloads     []struct {
			Name    string
			Payload string
		}
	}
}

// The Page struct holds page data for rendering
type Page struct {
	Config   Config
	Messages map[string][]pubsub.Message
}

// The PublishResponse struct holds responses to message publications
type PublishResponse struct {
	ID    string `json:"id"`
	Topic string `json:"topic"`
}

func main() {
	page = Page{}
	config := Config{}

	sseServer = sse.NewServer(nil)
	defer sseServer.Shutdown()

	box = packr.NewBox("./static")
	htmlTemplate, err := box.FindString("templates/main.html")
	if err != nil {
		log.Fatal(err)
	}
	tmpl = template.Must(template.New("t").Parse(htmlTemplate))

	maxMessages, _ = strconv.Atoi(getEnvDefault("GOPUBSUB_MAX_MESSAGES", "10"))

	configPath := getEnvDefault("GOPUBSUB_CONFIG", "config.yaml")
	configData, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Fatal(err)
	}
	configErr := yaml.Unmarshal([]byte(configData), &config)
	if configErr != nil {
		log.Fatal(configErr)
	}

	ctx := context.Background()

	clients = make(map[string]*pubsub.Client)

	messages = make(map[string][]pubsub.Message)
	page.Messages = messages

	topics = make(map[string]*pubsub.Topic)
	subscriptions = make(map[string]*pubsub.Subscription)

	for i, topicConfig := range config.Topics {
		client, err = pubsub.NewClient(ctx, topicConfig.Project)
		if err != nil {
			log.Fatal(err)
		}

		topicName = topicConfig.Name
		topicProject = topicConfig.Project
		if topicProject == "" {
			defaultProject := getEnvDefault("GOOGLE_CLOUD_PROJECT", "default-project")
			log.Fatal(fmt.Sprintf("Missing project for topic [%s], using default [%s]", topicName, defaultProject))
		}
		topicConfig.ID = fmt.Sprintf("%s/%s", topicProject, topicName)
		topicID := topicConfig.ID
		clients[topicID] = client
		topics[topicID] = clients[topicID].Topic(topicName)

		exists, err := topics[topicID].Exists(ctx)
		if err != nil {
			log.Fatal(err)
		}
		if !exists {
			log.Printf("Topic %v (project %s) doesn't exist - creating it", topicName, topicProject)
			_, err = clients[topicID].CreateTopic(ctx, topicName)
			if err != nil {
				log.Fatal(err)
			}
		}

		subscriptionName = topicConfig.Subscription
		if subscriptionName == "" {
			subscriptionName = fmt.Sprintf("sub-%s-%s", topicProject, topicName)
			log.Printf("No subscription name given for topic %s (project %s), using %s", topicName, topicProject, subscriptionName)
		}
		subscriptions[subscriptionName] = clients[topicID].Subscription(subscriptionName)
		subExists, err := subscriptions[subscriptionName].Exists(ctx)
		if err != nil {
			log.Fatal(err)
		}
		if !subExists {
			log.Printf("Subscription %v doesn't exist - creating it", subscriptionName)
			subConfig := pubsub.SubscriptionConfig{
				Topic: topics[topicID],
			}
			_, err = clients[topicID].CreateSubscription(ctx, subscriptionName, subConfig)
			if err != nil {
				log.Fatal(err)
			}
		}

		config.Topics[i] = topicConfig

		go pullMessages(ctx, subscriptions[subscriptionName], topics[topicID], topicProject)
	}

	page.Config = config

	http.HandleFunc("/", listHandler)
	http.HandleFunc("/publish", publishHandler)
	http.HandleFunc("/messages", messagesHandler)
	http.Handle("/events/", sseServer)

	fs := http.FileServer(box)
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

func pullMessages(ctx context.Context, subscription *pubsub.Subscription, topic *pubsub.Topic, project string) {
	cctx, _ := context.WithCancel(ctx)
	err := subscription.Receive(cctx, func(ctx context.Context, msg *pubsub.Message) {
		msg.Ack()
		log.Printf("Project [%s], topic [%s], subscription [%s], got message: %q, with attributes: %v\n", project, topic.ID(), subscription.ID(), string(msg.Data), msg.Attributes)

		topicID := fmt.Sprintf("%s/%s", project, topic.ID())
		messages[topicID] = append([]pubsub.Message{*msg}, messages[topicID]...)

		messagesMu.Lock()
		defer messagesMu.Unlock()
		if len(messages[topicID]) > maxMessages {
			messages[topicID] = messages[topicID][:maxMessages]
		}

		sseJSON, _ = json.Marshal(messages)
		sseServer.SendMessage("/events/messages", sse.SimpleMessage(string(sseJSON)))
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

	publishTopic := topics[r.FormValue("topic")]
	serverID, err := publishTopic.Publish(ctx, msg).Get(ctx)
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not publish message: %v", err), 500)
		return
	}

	response := PublishResponse{
		ID:    serverID,
		Topic: publishTopic.ID(),
	}

	jsonResponse, _ := json.Marshal(response)
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResponse)
}
