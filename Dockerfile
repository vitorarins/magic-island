FROM golang:1.12.9-stretch as builder
RUN apt-get update && \
    apt-get dist-upgrade -y && \
    apt-get install -y --no-install-recommends ca-certificates tzdata && \
	update-ca-certificates
WORKDIR /go/src/github.com/vitorarins/magic-island/
COPY . .
ENV GO111MODULE=on
RUN go mod download
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -installsuffix cgo -o app .

FROM scratch
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
WORKDIR /app/
COPY --from=builder /go/src/github.com/vitorarins/magic-island/action-data ./action-data/
COPY --from=builder /go/src/github.com/vitorarins/magic-island/app .

ENTRYPOINT ["./app"]
