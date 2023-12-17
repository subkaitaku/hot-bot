package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/subkaitaku/hotentry/hatebu"
)

func main() {
	fmt.Println("listen and serve on :8080")
	http.HandleFunc("/", hatebu.RenderHotentry)
	err := http.ListenAndServe("127.0.0.1:8080", nil)

	if err != nil {
		fmt.Printf("error: %v", err)
		os.Exit(1)
	}
}
