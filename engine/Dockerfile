FROM debian:jessie
RUN apt-get update && \
    apt-get install -y curl wget git ca-certificates && \
    mkdir /app && cd /app && \
    LAST_RELEASE=$(curl -s https://api.github.com/repos/ovh/cds/releases | grep tag_name | head -n 1 | cut -d '"' -f 4) && \
    curl -s https://api.github.com/repos/ovh/cds/releases | grep ${LAST_RELEASE} | grep browser_download_url | grep -E 'linux-amd64|worker|sql.tar.gz' | cut -d '"' -f 4 > files && \
    cat files | sort | uniq > filesToDownload && \
    while read f; do wget $f; done < filesToDownload && \
    chmod +x cds-engine-linux-amd64 && \
    cp cds-worker-linux-amd64 worker && \
    chmod +x worker && \
    tar xzf sql.tar.gz && \
    chown -R nobody:nogroup /app && \
    rm -rf /var/lib/apt/lists/*
USER nobody
WORKDIR /app
CMD ["/app/cds-engine-linux-amd64"]
