FROM debian:jessie
RUN apt-get update && \
    apt-get install -y curl wget ca-certificates && \
    mkdir /app && cd /app && \
    LAST_RELEASE=$(curl -s https://api.github.com/repos/ovh/cds/releases | grep tag_name | head -n 1 | cut -d '"' -f 4) && \
    curl -s https://api.github.com/repos/ovh/cds/releases | grep ${LAST_RELEASE} | grep browser_download_url | grep 'cds-worker-linux-amd64' | cut -d '"' -f 4 | xargs wget && \
    chmod +x cds-worker-linux-amd64 && \
    chown -R nobody:nogroup /app && \
    rm -rf /var/lib/apt/lists/*
USER nobody
WORKDIR /app
CMD ["/app/cds-worker-linux-amd64"]
