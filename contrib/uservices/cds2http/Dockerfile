FROM debian:jessie
RUN apt-get update && apt-get install -y ca-certificates
COPY ./service /app/service
RUN chmod +x /app/service && chown -R nobody:nogroup /app/service
USER nobody
CMD ["/app/service"]
