package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"../../pkg/superexporter"
)

//	"../../pkg/superexporter"

func main() {
	dispatcher := superexporter.NewDispatcher()

	http.HandleFunc("/scrape", dispatcher.Handler)
	srv := &http.Server{Addr: ":9292"}
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Println(err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
	<-sig
	dispatcher.CleanupAll()
	if err := srv.Shutdown(context.TODO()); err != nil {
		log.Println(err)
	}
}
