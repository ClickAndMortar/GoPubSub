# Go Pub/Sub UI

This tool eases debugging Google Clouds Pub/Sub with a simple UI.

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

Open http://localhost:8080 after running GoPubSub.

### From binary

See releases page.

### With docker-compose

```bash
docker-compose up -d
```

### From source

```bash
go run main.go
```

## Build

```bash
make
```

Linux and macOS (Darwin) binaries will be available under the `bin/` directory.

## Improvements

* [ ] Output message attributes
* [ ] Live update
* [x] JSON pretty-printing
* [x] Working message publication
* [x] Samples for message publication

## Credits

This app is heavily inspired by [Google's sample](https://github.com/GoogleCloudPlatform/golang-samples/blob/master/appengine_flexible/pubsub/pubsub.go).
