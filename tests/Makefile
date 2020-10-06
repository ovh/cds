define MINIO_CONFIG
export HOSTNAME=$$(hostname)
export AWS_DEFAULT_REGION=us-east-1
export S3_BUCKET=cds-it
export MINIO_ACCESS_KEY=AKIAIOSFODNN7EXAMPLE
export MINIO_SECRET_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
export AWS_ENDPOINT_URL=http://$$(hostname):9000
endef

MINIO_CONTAINER_ID = $(shell docker ps -f name=minio1 -q)

UNAME_S := $(shell uname -s| tr A-Z a-z)
VERSION := $(if ${CDS_VERSION},${CDS_VERSION},snapshot)

clean:
	@rm -f $(MINIO_CONTAINER_ID) $(MINIO_RC)
	@docker kill minio1 || true && docker rm minio1 || true

minio_start: $(MINIO_RC)
	@if [ -z "$(MINIO_CONTAINER_ID)" ]; then \
		docker rm minio1 >/dev/null 2>&1 || true; \
		echo "starting minio container"; \
		docker run -d -p 9000:9000 --name minio1 \
  			-e "MINIO_ACCESS_KEY=AKIAIOSFODNN7EXAMPLE" \
  			-e "MINIO_SECRET_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY" minio/minio server /data; \
	fi; \
	$(info # Here are the minio configuration variables)
	$(info $(MINIO_CONFIG))

minio_reset_bucket:
	@docker run --rm --link minio1:minio --entrypoint sh minio/mc -c "\
		while ! nc -z minio 9000; do echo 'Wait minio to startup...' && sleep 0.1; done; \
		sleep 5 && \
		mc config host add myminio http://minio:9000 AKIAIOSFODNN7EXAMPLE wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY && \
		mc rm -r --force myminio/cds-it  || true && \
		mc mb myminio/cds-it || true \
		"


test-docker-compose:
	docker-compose up --no-recreate -d cds-db cds-cache elasticsearch smtpmocksrv && \
	sleep 5	&& \
	docker-compose up --no-recreate cds-migrate && \
	docker-compose up cds-prepare && \
	docker-compose up -d cds-api && \
	echo "waiting 8s to let cds-api start.." && sleep 8 && \
	echo "downloading cdsctl from http://localhost:8081/download/cdsctl/$(UNAME_S)/amd64?variant=nokeychain -o cdsctl" && \
	curl http://localhost:8081/download/cdsctl/$(UNAME_S)/amd64?variant=nokeychain -o cdsctl && \
	chmod +x cdsctl && \
	cdsctl version && \
	export CDSCTL=`realpath ./cdsctl` && \
	TOKEN_CMD=`docker logs cds_cds-prepare_1|grep TOKEN` && \
	$$TOKEN_CMD && \
	./test.sh smoke_api initialization && \
	docker-compose up -d cds-cdn cds-ui cds-hooks cds-hatchery-local cds-elasticsearch && \
	echo "waiting 5s to let services start.." && sleep 5 && \
	./test.sh smoke_services cli;
