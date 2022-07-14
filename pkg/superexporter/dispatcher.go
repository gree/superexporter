package superexporter

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

const (
	cleanupPeriodSeconds           = 60
	workerInactiveThresholdSeconds = 300
)

type WorkerInfo struct {
	worker        WorkerInterface
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
	level.Info(d.logger).Log("msg", "Cleanup All Workers")
	for _, wi := range d.workersInfo {
		d.removeWorkerInfo(wi)
	}
}

func (d *Dispatcher) Handler(w http.ResponseWriter, r *http.Request) {
	defer d.periodicCleanup()
	targetStr := r.URL.Query().Get("target")
	level.Debug(d.logger).Log("msg", fmt.Sprintf("target:%s", targetStr))

	parsedUrl, err := url.Parse("unix://" + targetStr)
	if err != nil {
		level.Error(d.logger).Log("msg", "Failed to url.Parse", "err", err)
		return
	}
	host, port, _ := net.SplitHostPort(parsedUrl.Host)
	kind := "memcached"
	level.Debug(d.logger).Log("msg", fmt.Sprintf("kind:%s host:%s port:%s", kind, host, port))
	target := Target{Host: host, Port: port, kind: kind}

	wi, ok := d.workersInfo[target.id()]
	if !ok {
		level.Info(d.logger).Log("msg", fmt.Sprintf("Create new worker for %s", target.id()))
		wi, err = d.addWorkerInfo(&target)
		if err != nil {
			level.Error(d.logger).Log("msg", "addWorkerInfo err", "err", err)
			return
		}
	}
	level.Debug(d.logger).Log("msg", "do request", "target", target.id())
	if err := wi.worker.Request(&w, r); err != nil {
		level.Error(d.logger).Log("msg", "Request err", "err", err)
		return
	}
	wi.lastRequestAt = time.Now()
}

func (d *Dispatcher) addWorkerInfo(t *Target) (*WorkerInfo, error) {
	var worker WorkerInterface
	var err error
	switch t.kind {
	case "memcached":
		worker, err = CreateMemcachedWorker(t, d.logger)
	default:
		return nil, errors.New("not supported")
	}
	if err != nil {
		level.Error(d.logger).Log("msg", "CreateWorker failed", "err", err)
		return nil, err
	}
	wi := &WorkerInfo{worker: worker, target: t}
	d.workersInfo[t.id()] = wi
	return wi, nil
}

func (d *Dispatcher) removeWorkerInfo(wi *WorkerInfo) {
	if err := wi.worker.Destroy(); err != nil {
		level.Error(d.logger).Log("msg", "DestoryWorker failed", "err", err)
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
		level.Debug(d.logger).Log("msg", "periodic cleanup")
		for _, wi := range d.workersInfo {
			if int(now.Sub(wi.lastRequestAt).Seconds()) >= workerInactiveThresholdSeconds {
				level.Info(d.logger).Log("msg", fmt.Sprintf("remove worker for %s", wi.target.id()))
				defer d.removeWorkerInfo(wi)
			}
		}
		d.lastCleanupAt = now
	}
}
