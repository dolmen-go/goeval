package main

import (
	"io"
	"log"
	"net/http"
	"os"
)

func main() {
	// TODO User-Agent
	resp, err := http.Post("https://go.dev/_/share", "text/plain; charset=utf-8", os.Stdin)
	if err != nil {
		log.Fatal("share:", err)
	}
	defer resp.Body.Close()
	id, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("share:", err)
	}
	io.WriteString(os.Stdout, "https://go.dev/play/p/"+string(id)+"\n")
}
