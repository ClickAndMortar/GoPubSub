FROM node:10-alpine as node

WORKDIR /tmp

COPY package.json package-lock.json webpack.config.js /tmp/
COPY assets /tmp/assets

RUN npm install && npm run build-prod

FROM golang:1.13 as go-builder

WORKDIR /go/src/app

COPY main.go go.mod go.sum ./
COPY static ./static

COPY --from=node /tmp/static/bundle.js ./static/bundle.js

RUN go get -d -v ./...
RUN go install -v ./...

FROM debian:buster-slim

COPY --from=go-builder /go/bin/GoPubSub /usr/local/bin/gopubsub
COPY --from=go-builder /go/src/app /go/src/app

# Legacy config directory
WORKDIR /go/src/app/

CMD ["gopubsub"]
