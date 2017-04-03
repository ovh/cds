FROM debian:jessie
RUN apt-get update && \
    apt-get install -y curl wget ca-certificates && \
    mkdir /app && cd /app && \
    curl -s https://api.github.com/repos/ovh/cds/releases | grep browser_download_url | grep "/hatchery" | head -n 1 | cut -d '"' -f 4 | xargs wget && \
    chmod +x hatchery && \
    chown -R nobody:nogroup /app && \
    rm -rf /var/lib/apt/lists/*
USER nobody
CMD ["/app/hatchery"]
