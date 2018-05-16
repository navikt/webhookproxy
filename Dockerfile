FROM golang:1.9 as builder

WORKDIR /go/src/github.com/navikt/webhookproxy
COPY . .

RUN go test -v ./...
RUN CGO_ENABLED=0 GOOS=linux go install -a -installsuffix cgo -v

FROM scratch
COPY --from=builder /go/bin/webhookproxy /
CMD ["/webhookproxy"]
