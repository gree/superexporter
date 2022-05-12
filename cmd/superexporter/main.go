package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"superexporter/pkg/superexporter"

	"github.com/go-kit/log/level"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/promlog/flag"
	"gopkg.in/alecthomas/kingpin.v2"
)

func main() {
	promlogConfig := &promlog.Config{}
	flag.AddFlags(kingpin.CommandLine, promlogConfig)
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()
	logger := promlog.New(promlogConfig)

	dispatcher := superexporter.NewDispatcher(logger)

	http.HandleFunc("/scrape", dispatcher.Handler)
	srv := &http.Server{Addr: ":9150"}
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			level.Error(logger).Log("msg", "ListenAndServe err", "err", err)
			os.Exit(1)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
	<-sig
	dispatcher.CleanupAll()
	if err := srv.Shutdown(context.TODO()); err != nil {
		level.Error(logger).Log("msg", "failed to shutdown server", "err", err)
	}
}
