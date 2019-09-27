# Go Pub/Sub UI

[![Go Report Card](https://goreportcard.com/badge/github.com/ClickAndMortar/GoPubSub)](https://goreportcard.com/report/github.com/ClickAndMortar/GoPubSub)

This tool eases debugging Google Clouds Pub/Sub with a simple UI.

Features include:

* Subscription to given topics
* Automatic creation of non-existent topics or subscriptions
* Publishing messages from UI to configured topics
* Configurable pre-defined payloads per topic
* Live update of received messages

![Screenshot](https://raw.githubusercontent.com/ClickAndMortar/GoPubSub/master/gopubsub.png)

## Configuration

Following environment variables may be set:

| Variable | Usage | Default value |
|---|---|---|
| `GOPUBSUB_CONFIG` | Config file path | `config.yaml` |
| `GOPUBSUB_PORT` | Listening HTTP port | `8080` |
| `GOPUBSUB_MAX_MESSAGES` | Only keep last _n_ messages per topic | `10` |
| `GOOGLE_APPLICATION_CREDENTIALS` | Path to JSON credentials file | _none_ |
| `PUBSUB_EMULATOR_HOST` | Host of emulator (see below) | _none_ |

### The `config.yaml` file

To get GoPubSub working, you'll need to create `config.yaml` file with your topics and subscriptions info:

```yaml
project: test
topics:
  -
    name: my-topic
    subscription: my-subscription
  -
    name: my-other-topic
    subscription: my-other-subscription
    payloads:
      -
        name: hello
        payload: |
          {"hello": "world"}
      -
        name: hello-again
        payload: |
          {"hello": "world again"}
  -
    name: my-last-topic
```

Note that all non-existant topic or subscription will be created. If no subscription is given for a topic, it will be created as `sub-<topic-name>`.

## Usage

### Using binary

#### Pre-built

Download latest release for your OS from the [releases](https://github.com/ClickAndMortar/GoPubSub/releases) page, or run:

```bash
go get -u github.com/clickandmortar/gopubsub
```

Then:

```bash
export GOPUBSUB_CONFIG="/path/to/config.yaml"

# To use local emulator
export PUBSUB_EMULATOR_HOST=localhost:8085

# To use actual Pub/Sub service
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/application_default_credentials.json"

gopubsub
```

And open http://localhost:8080.

### Using Docker image

```bash
docker pull clickandmortar/gopubsub
docker run -d \
 -v $(pwd)/config.yaml:/go/src/app/config.yaml \
 -p 8080:8080 \
 --name="gopubsub" \
 -e PUBSUB_EMULATOR_HOST=pubsub:8085 \
 clickandmortar/gopubsub
```

### With docker-compose

```bash
docker-compose up -d
```

### From source

```bash
go run main.go
```

## Build

### Binaries

```bash
make
```

Linux and macOS (Darwin) binaries will be available under the `bin/` directory.

### Docker image

```bash
make docker
```

## Improvements

* [ ] Remove initial AJAX call
* [ ] Output message attributes
* [ ] Use Vue.js for form
* [ ] Fetch available topics list
* [x] Live update (using SSE)
* [x] JSON pretty-printing
* [x] Working message publication
* [x] Samples for message publication

## Credits

This app is heavily inspired by [Google's sample](https://github.com/GoogleCloudPlatform/golang-samples/blob/master/appengine_flexible/pubsub/pubsub.go).
