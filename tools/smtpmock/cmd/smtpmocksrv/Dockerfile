# build stage
FROM golang:1.13.4 AS builder
ADD . $GOPATH/src/github.com/ovh/cds/tools/smtpmock/cmd/smtpmocksrv
WORKDIR $GOPATH/src/github.com/ovh/cds/tools/smtpmock/cmd/smtpmocksrv
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o /tmp/smtpmocksrv

# final stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /tmp/smtpmocksrv ./
RUN chmod +x ./smtpmocksrv
ENTRYPOINT ["./smtpmocksrv"]
EXPOSE 2023 2024
