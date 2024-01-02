package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/subkaitaku/hotentry/hatebu"
)

func main() {
	fmt.Println("listen and serve on :5000")
	http.HandleFunc("/", hatebu.RenderHotentry)
	http.HandleFunc("/register", hatebu.RegisterBlock)
	err := http.ListenAndServe("127.0.0.1:5000", nil)

	if err != nil {
		fmt.Printf("error: %v", err)
		os.Exit(1)
	}
}
