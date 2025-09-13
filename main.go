package main

import (
	"log"
	"net/http"
	"strconv"

	"github.com/iwanhae/coview/internal/config"
	"github.com/iwanhae/coview/internal/server"
)

func main() {
	if err := config.Load("config/config.yaml"); err != nil {
		log.Fatal("Failed to load config: ", err)
	}

	cfg := config.Get()
	portStr := ":" + strconv.Itoa(cfg.Server.Port)

	http.HandleFunc("/", server.Handler)

	log.Printf("Server starting on %s", portStr)
	log.Fatal(http.ListenAndServe(portStr, nil))
}
