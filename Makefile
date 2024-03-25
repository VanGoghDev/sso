migration:
		go run ./cmd/migrator --storage-path=./storage/sso.db --migrations-path=./migrations
run:
		go run cmd/sso/main.go --config=./config/local_tests.yaml