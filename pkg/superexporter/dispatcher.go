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
	host string
	port string
	kind string
}

func (t *Target) id() string {
	return t.host + "-" + t.port
}

type WorkerInfo struct {
	//kind   string
	worker        *Worker
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
		if err := DestoryWorker(wi.worker); err != nil {
			log.Print("Cleanup Error!!!!!", err)
		}
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
	log.Print("kind:", kind, "host:", host, " port:", port)
	target := Target{host: host, port: port, kind: kind}

	wi, ok := srv.workersInfo[target.id()]
	if !ok {
		log.Print("create new worker!")
		worker, err := CreateWorker(&target)
		if err != nil {
			log.Print("err!!!", err)
			return
		}
		wi = &WorkerInfo{worker: worker}
		srv.workersInfo[target.id()] = wi
	}
	log.Print("let's request")
	if err := wi.worker.Request(&w, r); err != nil {
		return
	}
	wi.lastRequestAt = time.Now()
}

func initSigHandler() {
	// TODO: すでにハンドラ登録してあったらなんやかんや
	waits := make(chan os.Signal, 1)
	signal.Notify(waits, syscall.SIGCHLD)

	go func() {
		for {
			sig := <-waits
			fmt.Println("handle sigchld:", sig)
		}
	}()
}

func (srv *Dispatcher) periodicCleanup() {
	log.Print("yeeey, periodic cleanup ")
	now := time.Now()
	if int(now.Sub(srv.lastCleanupAt).Seconds()) >= cleanupPeriodSeconds {
		for _, wi := range srv.workersInfo {
			if int(now.Sub(wi.lastRequestAt).Seconds()) >= workerInactiveThresholdSeconds {
				defer DestoryWorker(wi.worker)
			}
		}
		srv.lastCleanupAt = now
	}
}
