FROM golang:1.13

WORKDIR /go/src/app

COPY main.go template.html static ./

RUN go get -d -v ./...
RUN go install -v ./...

CMD ["app"]
