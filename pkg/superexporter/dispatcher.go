package superexporter

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-kit/log"
)

const (
	cleanupPeriodSeconds           = 60
	workerInactiveThresholdSeconds = 300
)

type WorkerInfo struct {
	worker        *Worker
	target        *Target
	lastRequestAt time.Time
}

type Dispatcher struct {
	workersInfo   map[string]*WorkerInfo
	lastCleanupAt time.Time
	logger        log.Logger
}

func NewDispatcher(logger log.Logger) *Dispatcher {
	initSigHandler()
	wi := map[string]*WorkerInfo{}
	return &Dispatcher{workersInfo: wi, logger: logger}
}

func (d *Dispatcher) CleanupAll() {
	d.logger.Log("severity", "INFO", "msg", "Cleanup All Workers")
	for _, wi := range d.workersInfo {
		d.removeWorkerInfo(wi)
	}
}

func (d *Dispatcher) Handler(w http.ResponseWriter, r *http.Request) {
	defer d.periodicCleanup()
	targetStr := r.URL.Query().Get("target")
	fmt.Println("target:", targetStr)

	parsedUrl, err := url.Parse("unix://" + targetStr)
	if err != nil {
		d.logger.Log("severity", "ERROR", "err", err)
	}
	host, port, _ := net.SplitHostPort(parsedUrl.Host)
	kind := "memcached"
	d.logger.Log("severity", "INFO", "msg", fmt.Sprintf("kind:%s host:%s port:%s", kind, host, port))
	target := Target{Host: host, Port: port, kind: kind}

	wi, ok := d.workersInfo[target.id()]
	if !ok {
		d.logger.Log("severity", "INFO", "msg", "create new worker")
		wi, err = d.addWorkerInfo(&target)
		if err != nil {
			d.logger.Log("severity", "ERROR", "err", err)
			return
		}
	}
	d.logger.Log("severity", "INFO", "msg", "do request")
	if err := wi.worker.Request(&w, r); err != nil {
		d.logger.Log("severity", "ERROR", "err", err)
		return
	}
	wi.lastRequestAt = time.Now()
}

func (d *Dispatcher) addWorkerInfo(t *Target) (*WorkerInfo, error) {
	worker, err := CreateWorker(t, d.logger)
	if err != nil {
		d.logger.Log("severity", "ERROR", "err", err)
		return nil, err
	}
	wi := &WorkerInfo{worker: worker, target: t}
	d.workersInfo[t.id()] = wi
	return wi, nil
}

func (d *Dispatcher) removeWorkerInfo(wi *WorkerInfo) {
	if err := DestoryWorker(wi.worker); err != nil {
		d.logger.Log("severity", "ERROR", "err", err, "msg", "DestoryWorker Error!")
	}
	delete(d.workersInfo, wi.target.id())
}

func initSigHandler() {
	// TODO: すでにハンドラ登録してあったらなんやかんや
	waits := make(chan os.Signal, 1)
	signal.Notify(waits, syscall.SIGCHLD)

	go func() {
		for {
			sig := <-waits
			syscall.Wait4(-1, nil, syscall.WNOHANG|syscall.WUNTRACED, nil)
			fmt.Println("handle sigchld:", sig)
		}
	}()
}

func (d *Dispatcher) periodicCleanup() {
	now := time.Now()
	if int(now.Sub(d.lastCleanupAt).Seconds()) >= cleanupPeriodSeconds {
		d.logger.Log("severity", "INFO", "msg", "periodic cleanup")
		for _, wi := range d.workersInfo {
			if int(now.Sub(wi.lastRequestAt).Seconds()) >= workerInactiveThresholdSeconds {
				defer d.removeWorkerInfo(wi)
			}
		}
		d.lastCleanupAt = now
	}
}
