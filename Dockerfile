FROM golang:alpine as app-builder
WORKDIR /go/src/app
COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o pinub cmd/pinub/main.go

FROM registry.access.redhat.com/ubi9/ubi-micro:9.1.0-17
COPY --from=app-builder /go/src/app/pinub /pinub
EXPOSE 8080
ENTRYPOINT ["/pinub"]
