FROM debian:jessie
RUN apt-get update && \
    apt-get install -y curl wget ca-certificates && \
    mkdir /app && cd /app && \
    LAST_RELEASE=$(curl -s https://api.github.com/repos/ovh/cds/releases | grep tag_name | head -n 1 | cut -d '"' -f 4) && \
    curl -s https://api.github.com/repos/ovh/cds/releases | grep ${LAST_RELEASE} | grep browser_download_url | grep 'ui.tar.gz' | cut -d '"' -f 4 | xargs wget && \
    tar xzf ui.tar.gz && mv dist/* . && \
    wget https://caddyserver.com/download/linux/amd64 && mv amd64 caddy.tar.gz && \
    tar xzf caddy.tar.gz && \
    chmod +rx caddy setup && \
    chown -R nobody:nogroup /app && \
    rm -rf /var/lib/apt/lists/*
USER nobody
WORKDIR /app
CMD ["/app/setup"]
