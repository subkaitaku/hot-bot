package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/subkaitaku/hotentry/hatebu"
)

func main() {
	fmt.Println("listen and serve on :8080")
	err := http.ListenAndServe("127.0.0.1:8080", nil)
	http.HandleFunc("/", hatebu.RenderHotentry)
	if err != nil {
		fmt.Printf("error: %v", err)
		os.Exit(1)
	}
}
