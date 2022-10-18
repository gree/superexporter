package superexporter

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	workersNum = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "superexporter",
		Name:      "num_workers",
		Help:      "Current number of valid worker processes.",
	})
)

func (d *Dispatcher) RecordMetrics() {
	go func() {
		for {
			workersNum.Set(float64(len(d.workersInfo)))

			time.Sleep(5 * time.Second)
		}
	}()
}
