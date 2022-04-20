package main

import (
	"context"
	stdlog "log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"superexporter/pkg/superexporter"

	kitlog "github.com/go-kit/log"
)

func main() {
	logger := kitlog.NewJSONLogger(kitlog.NewSyncWriter(os.Stdout))
	dispatcher := superexporter.NewDispatcher(logger)

	http.HandleFunc("/scrape", dispatcher.Handler)
	srv := &http.Server{Addr: ":9150"}
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			stdlog.Fatal(err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
	<-sig
	dispatcher.CleanupAll()
	if err := srv.Shutdown(context.TODO()); err != nil {
		stdlog.Println(err)
	}
}
