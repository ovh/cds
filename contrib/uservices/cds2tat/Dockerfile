FROM debian:jessie
RUN apt-get update && apt-get install -y ca-certificates
COPY ./api /app/api
RUN chmod +x /app/api && chown -R nobody:nogroup /app/api
USER nobody
CMD ["/app/api"]
