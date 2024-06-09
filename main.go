package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/subkaitaku/hotentry/hatebu"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
			port = "8080"
	}
	fmt.Println("listen and serve on :" + port)
	http.HandleFunc("/", hatebu.RenderHotentry)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		fmt.Printf("error: %v", err)
		os.Exit(1)
	}
}
