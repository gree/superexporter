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

type Target struct {
	Host string
	Port string
	kind string
}

func (t *Target) id() string {
	return t.Host + "-" + t.Port
}

type WorkerInfo struct {
	//kind   string
	worker        *Worker
	target        *Target
	lastRequestAt time.Time
}

type Dispatcher struct {
	//workerCmd   []string
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

func (srv *Dispatcher) Handler(w http.ResponseWriter, r *http.Request) {
	defer srv.periodicCleanup()
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

	wi, ok := srv.workersInfo[target.id()]
	if !ok {
		log.Print("create new worker")
		wi, err = srv.addWorkerInfo(&target)
		if err != nil {
			log.Print("err:", err)
			return
		}
		/* worker, err := CreateWorker(&target)
		if err != nil {
			log.Print("err:", err)
			return
		}
		wi = &WorkerInfo{worker: worker}
		srv.workersInfo[target.id()] = wi */
	}
	log.Print("do request")
	if err := wi.worker.Request(&w, r); err != nil {
		log.Print(err)
		return
	}
	wi.lastRequestAt = time.Now()
}

func (srv *Dispatcher) addWorkerInfo(t *Target) (*WorkerInfo, error) {
	worker, err := CreateWorker(t)
	if err != nil {
		log.Print("err:", err)
		return nil, err
	}
	wi := &WorkerInfo{worker: worker, target: t}
	srv.workersInfo[t.id()] = wi
	return wi, nil
}

func (srv *Dispatcher) removeWorkerInfo(wi *WorkerInfo) {
	if err := DestoryWorker(wi.worker); err != nil {
		log.Print("DestoryWorker Error!", err)
	}
	delete(srv.workersInfo, wi.target.id())
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

func (srv *Dispatcher) periodicCleanup() {
	now := time.Now()
	if int(now.Sub(srv.lastCleanupAt).Seconds()) >= cleanupPeriodSeconds {
		log.Print("periodic cleanup")
		for _, wi := range srv.workersInfo {
			if int(now.Sub(wi.lastRequestAt).Seconds()) >= workerInactiveThresholdSeconds {
				//defer DestoryWorker(wi.worker)
				defer srv.removeWorkerInfo(wi)
			}
		}
		srv.lastCleanupAt = now
	}
}
