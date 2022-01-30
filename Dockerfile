FROM golang:alpine as app-builder
WORKDIR /go/src/app
COPY . .
RUN CGO_ENABLED=0 go build .

FROM scratch
COPY --from=app-builder /go/src/app/pinub /pinub
ENTRYPOINT ["/pinub"]
