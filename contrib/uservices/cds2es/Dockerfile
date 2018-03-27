FROM debian:jessie
RUN apt-get update && apt-get install -y ca-certificates
COPY ./cds2er /app/cds2es
COPY ./config.toml /app/config.toml
RUN chmod +x /app/cds2es && chown nobody:nogroup /app/cds2es
USER nobody
CMD ["/app/cds2es -f config.toml"]
