package main

import (
	"flag"
	"log"
	"net/http"
)

var addr = flag.String("addr", ":8000", "")

func main() {
	flag.Parse()

	log.Println("Listening on", *addr)
	log.Fatal(http.ListenAndServe(*addr, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// fmt.Fprintf(w, "Hello, %s!", r.UserAgent())
	})))
}
