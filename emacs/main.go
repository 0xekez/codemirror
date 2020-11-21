// Echo's all requests to stdout wherever it is running.
package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
)

func echo(w http.ResponseWriter, r *http.Request) {
	requestDump, err := httputil.DumpRequest(r, true)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(requestDump))
}

func main() {
	http.HandleFunc("/", echo)

	fmt.Printf("Starting server for testing HTTP POST...\n")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
