# Go Pub/Sub UI

This tool eases debugging Google Clouds Pub/Sub with a simple UI.

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
```

Note that all non-existant topic or subscription will be created.

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

[ ] JSON pretty-printing
[ ] Live update
[ ] Working message publication
[ ] Samples for message publication

## Credits

This app is heavily inspired by [Google's sample](https://github.com/GoogleCloudPlatform/golang-samples/blob/master/appengine_flexible/pubsub/pubsub.go).
