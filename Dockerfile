FROM alpine:3.16
RUN apk update && apk update && \
    apk --no-cache add curl && \
    apk --no-cache add gpg && \
    apk --no-cache add git && \
    apk --no-cache add tzdata && \
    apk --no-cache add openssh-client && \
    apk --no-cache add ca-certificates && rm -rf /var/cache/apk/* 
RUN update-ca-certificates
RUN mkdir -p /app/sql /app/ui_static_files
COPY dist/cds-engine-* /app/
COPY dist/cdsctl-* /app/
COPY dist/cds-worker-* /app/
COPY dist/sql.tar.gz /app/
COPY dist/ui.tar.gz /app/
COPY dist/cds-docs.tar.gz /app/

RUN addgroup cds && adduser cds -G cds -D
RUN chmod +x /app/cds-engine-linux-amd64 && \
    tar xzf /app/sql.tar.gz -C /app/sql && \
    tar xzf /app/ui.tar.gz -C /app/ui_static_files && \
    tar xzf /app/cds-docs.tar.gz -C /app/ui_static_files && \
    mv /app/ui_static_files/cds-docs /app/ui_static_files/docs && \
    chown -R cds:cds /app
USER cds
WORKDIR /app
CMD ["/app/cds-engine-linux-amd64"]
