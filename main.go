package main

import (
	"log"
	"net/http"

	"github.com/GuilhermeCaruso/bellt"
	"github.com/dimitrovvlado/pglock/db"
	"github.com/dimitrovvlado/pglock/handler"
)

func main() {
	//TODO extract configuration as flags/file
	db.DefaultTTL = 5 //setting this for testing purposes
	err := db.Init("postgres://pglock:pglock@postgres:5432/pglock?sslmode=disable", 10)
	if err != nil {
		log.Fatal(err)
	}

	//Migrate database schema
	err = db.MigrateDatabase()
	if err != nil {
		log.Fatal(err)
	}

	router := bellt.NewRouter()
	router.HandleGroup("/v1",
		router.SubHandleFunc("/lock", handler.LockHandle, "POST", "DELETE"),
	)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
