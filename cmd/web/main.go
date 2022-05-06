package main

import (
	"log"
	"net/http"
)

func main() {
	routes := routes()

	log.Println("Starting web server on port 8080")
	err := http.ListenAndServe(":8080", routes)
	if err != nil {
		log.Fatalln(err)
	}
}
