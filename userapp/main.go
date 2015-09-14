package main

import (
	"log"
	"net/http"
)

func main() {
	http.Handle("/", http.FileServer(http.Dir("")))
	log.Fatal(http.ListenAndServe(":8081", nil))
}
