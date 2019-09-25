# Go Pub/Sub UI

This tool eases debugging Google Clouds Pub/Sub with a simple UI.

![Screenshot](https://raw.githubusercontent.com/ClickAndMortar/GoPubSub/master/gopubsub.png)

## Configuration

Define the topics and associated subscriptions in `config.yaml`:

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

```bash
go run main.go
```

You may specify listening port with `GOPUBSUB_PORT` environment variable (defaults to `8080`).

### With emulator

```
docker-compose up -d
```

And run this app:

```
PUBSUB_EMULATOR_HOST=localhost:8085 go run main.go
```

## Improvements

* [x] JSON pretty-printing
* [ ] Live update
* [x] Working message publication
* [x] Samples for message publication

## Credits

This app is heavily inspired by [Google's sample](https://github.com/GoogleCloudPlatform/golang-samples/blob/master/appengine_flexible/pubsub/pubsub.go).
