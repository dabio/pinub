FROM golang:alpine as app-builder
WORKDIR /go/src/app
COPY . .
RUN CGO_ENABLED=0 go build -o pinub cmd/server/main.go

FROM scratch
COPY --from=app-builder /go/src/app/pinub /pinub
ENTRYPOINT ["/pinub"]
