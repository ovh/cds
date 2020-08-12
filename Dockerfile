FROM debian:buster
RUN apt-get update && \
    apt-get install -y curl ca-certificates telnet dnsutils gpg git && \
    mkdir -p /app/sql /app/ui_static_files /app/panic_dumps
COPY dist/cds-engine-* /app/
COPY dist/cdsctl-* /app/
COPY dist/cds-worker-* /app/
COPY dist/sql.tar.gz /app/
COPY dist/ui.tar.gz /app/
COPY dist/cds-docs.tar.gz /app/

RUN groupadd -r cds && useradd --create-home -r -g cds cds
RUN chmod +w /app/panic_dumps && \
    chmod +x /app/cds-engine-linux-amd64 && \
    tar xzf /app/sql.tar.gz -C /app/sql && \
    tar xzf /app/ui.tar.gz -C /app/ui_static_files && \
    tar xzf /app/cds-docs.tar.gz -C /app/ui_static_files && \
    mv /app/ui_static_files/cds-docs /app/ui_static_files/docs && \
    chown -R cds:cds /app
USER cds
WORKDIR /app
CMD ["/app/cds-engine-linux-amd64"]
