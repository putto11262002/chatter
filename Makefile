source_env:
	@export $(shell cat .env | xargs)

migration: source_env
	GOOSE_DRIVER=sqlite3 GOOSE_DBSTRING=$(DB_PATH) GOOSE_MIGRATION_DIR=migrations goose $(ARGS)

dev: source_env
	@make -j2 dev_server dev_client

dev_server:
	@air

dev_client:
	@bun dev

build_client: 
	@bun run build

build_server:
	@go build -o bin/chatter main.go

build: build_client build_server

