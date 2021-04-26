package main

import (
	"log"
	"net/http"
)

func main() {
	http.Handle("/", http.FileServer(http.Dir("e2e/broadcast/static")))
	log.Fatal(http.ListenAndServe(":7070", nil))
}
