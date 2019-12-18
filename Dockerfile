FROM debian:buster
RUN apt-get update && \
    apt-get install -y curl ca-certificates telnet dnsutils gpg git && \
    mkdir -p /app/sql /app/ui_static_files /app/panic_dumps
COPY dist/* /app/
RUN groupadd -r cds && useradd -r -g cds cds
RUN chmod +w /app/panic_dumps && \
    chmod +x /app/cds-engine-* && \
    tar xzf /app/sql.tar.gz -C /app/sql && \
    tar xzf /app/ui.tar.gz -C /app/ui_static_files && \
    chown -R cds:cds /app
USER cds
WORKDIR /app
CMD ["/app/cds-engine-linux-amd64"]
