# build stage
FROM golang:1.17 AS builder
ADD . $GOPATH/src/github.com/ovh/cds/tools/smtpmock
WORKDIR $GOPATH/src/github.com/ovh/cds/tools/smtpmock
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o /tmp/smtpmocksrv github.com/ovh/cds/tools/smtpmock/server
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o /tmp/smtpmockcli github.com/ovh/cds/tools/smtpmock/cli

# final stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /tmp/smtpmocksrv ./
COPY --from=builder /tmp/smtpmockcli ./
RUN chmod +x ./smtpmocksrv
ENTRYPOINT ["./smtpmocksrv"]
CMD ["start"]
EXPOSE 2023 2024
