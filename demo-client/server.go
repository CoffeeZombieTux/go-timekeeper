package main

import (
	"flag"
	"log"
	"net/http"
)

func main() {
	addr := flag.String("addr", ":5173", "server listen address")
	dir := flag.String("dir", ".", "directory to serve")
	flag.Parse()

	log.Printf("Serving %s at http://localhost%s\n", *dir, *addr)
	if err := http.ListenAndServe(*addr, http.FileServer(http.Dir(*dir))); err != nil {
		log.Fatal(err)
	}
}
