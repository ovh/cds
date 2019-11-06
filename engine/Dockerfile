FROM debian:buster
RUN apt-get update && \
    apt-get install -y curl wget git ca-certificates && \
    mkdir /app && cd /app && \
    LAST_RELEASE=$(curl -s https://api.github.com/repos/ovh/cds/releases | grep tag_name | head -n 1 | cut -d '"' -f 4) && \
    curl -s https://api.github.com/repos/ovh/cds/releases | grep ${LAST_RELEASE} | grep browser_download_url | grep -E 'cdsctl|engine|worker|sql.tar.gz|ui.tar.gz' | cut -d '"' -f 4 > files && \
    cat files | sort | uniq > filesToDownload && \
    while read f; do wget $f; done < filesToDownload && \
    chmod +x cds* && \
    cp cds-worker-linux-amd64 worker && \
    tar xzf sql.tar.gz && \
    mkdir /app/panic_dumps /var/lib/cds-engine && \
    chown -R nobody:nogroup /app /var/lib/cds-engine && \
    chmod +w /app/panic_dumps && \
    chmod 700 /var/lib/cds-engine && \
    mkdir -p /app/ui_static_files && cd /app && tar xzf ui.tar.gz -C ui_static_files && \
    chown -R nobody:nogroup /app/ui_static_files && \
    rm -rf /var/lib/apt/lists/*
USER nobody
WORKDIR /app
CMD ["/app/cds-engine-linux-amd64"]
