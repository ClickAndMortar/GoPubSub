FROM node:10-alpine as node

WORKDIR /tmp

COPY package.json package-lock.json webpack.config.js /tmp/
COPY assets /tmp/assets

RUN npm install && npm run build-prod

FROM golang:1.13

WORKDIR /go/src/app

COPY main.go ./
COPY static ./static

COPY --from=node /tmp/static/bundle.js ./static/bundle.js

RUN go get -d -v ./...
RUN go install -v ./...

CMD ["app"]
