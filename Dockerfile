FROM golang:alpine as app-builder
WORKDIR /go/src/app
COPY . .
RUN CGO_ENABLED=0 go build -o pinub cmd/server/main.go

FROM registry.access.redhat.com/ubi9/ubi-micro:9.0.0-24
COPY --from=app-builder /go/src/app/pinub /pinub
EXPOSE 8080
ENTRYPOINT ["/pinub"]
