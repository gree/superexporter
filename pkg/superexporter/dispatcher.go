package superexporter

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"
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
}

func NewDispatcher() *Dispatcher {
	initSigHandler()
	wi := map[string]*WorkerInfo{}
	return &Dispatcher{workersInfo: wi}
}

func (d *Dispatcher) CleanupAll() {
	log.Println("Cleanup All Workers")
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
		log.Print(err)
	}
	host, port, _ := net.SplitHostPort(parsedUrl.Host)
	kind := "memcached"
	log.Print("kind:", kind, " host:", host, " port:", port)
	target := Target{Host: host, Port: port, kind: kind}

	wi, ok := d.workersInfo[target.id()]
	if !ok {
		log.Print("create new worker")
		wi, err = d.addWorkerInfo(&target)
		if err != nil {
			log.Print("err:", err)
			return
		}
	}
	log.Print("do request")
	if err := wi.worker.Request(&w, r); err != nil {
		log.Print(err)
		return
	}
	wi.lastRequestAt = time.Now()
}

func (d *Dispatcher) addWorkerInfo(t *Target) (*WorkerInfo, error) {
	worker, err := CreateWorker(t)
	if err != nil {
		log.Print("err:", err)
		return nil, err
	}
	wi := &WorkerInfo{worker: worker, target: t}
	d.workersInfo[t.id()] = wi
	return wi, nil
}

func (d *Dispatcher) removeWorkerInfo(wi *WorkerInfo) {
	if err := DestoryWorker(wi.worker); err != nil {
		log.Print("DestoryWorker Error!", err)
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
		log.Print("periodic cleanup")
		for _, wi := range d.workersInfo {
			if int(now.Sub(wi.lastRequestAt).Seconds()) >= workerInactiveThresholdSeconds {
				defer d.removeWorkerInfo(wi)
			}
		}
		d.lastCleanupAt = now
	}
}
