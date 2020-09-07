.PHONY: build clean lint test

build: sql.tar.gz
sql.tar.gz:
	@tar czf sql.tar.gz api cdn

clean:
	@rm -f sql.tar.gz missing_pk.log

lint:
	@./lint.sh api
	@./lint.sh cdn

TEST_DB_HOST = $(if ${CDS_DATABASE_HOST},${CDS_DATABASE_HOST},localhost)
TEST_DB_PORT = $(if ${CDS_DATABASE_PORT},${CDS_DATABASE_PORT},5432)
TEST_DB_USER = $(if ${CDS_DATABASE_USER},${CDS_DATABASE_USER},cds)
TEST_DB_PASS = $(if ${CDS_DATABASE_PASS},${CDS_DATABASE_PASS},cds)
TEST_DB_NAME = $(if ${CDS_DATABASE_NAME},${CDS_DATABASE_NAME},cds)

test:
	@./test.sh ${TEST_DB_USER} ${TEST_DB_PASS} ${TEST_DB_NAME} ${TEST_DB_HOST} ${TEST_DB_PORT}
