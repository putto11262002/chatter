source_env:
	@export	$(shell cat .env | xargs)

migration: source_env
	GOOSE_DRIVER=sqlite3 GOOSE_DBSTRING=$(DB_PATH) GOOSE_MIGRATION_DIR=migrations goose $(ARGS)


