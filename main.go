package main

import chatter "github.com/putto11262002/chatter/app"

func main() {
	loader := chatter.DefaultConfigLoader{}
	defaultConfig, err := loader.Load()
	if err != nil {
		panic(err)
	}
	defaultConfig.SQLiteFile = "./chatter.db"
	defaultConfig.MigrationDir = "./migrations"
	app := chatter.New(defaultConfig)
	app.Start()
}
