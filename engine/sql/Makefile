.PHONY: build clean lint test

build: sql.tar.gz

sql.tar.gz:
	@tar czf sql.tar.gz *.sql

clean:
	@rm -f sql.tar.gz missing_pk.log

lint:
	@./lint.sh

test:
	@./tests.sh