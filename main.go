package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/subkaitaku/hot-bot/hatebu"
)

func main() {
	http.HandleFunc("/", hatebu.PrintHatebu)
	fmt.Println("listen and serve on :8080")
	err := http.ListenAndServe("127.0.0.1:8080", nil)
	if err != nil {
		fmt.Printf("error: %v", err)
		os.Exit(1)
	}
}
