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
)

type Target struct {
	host string
	port string
}

func (t *Target) id() string {
	return t.host + "-" + t.port
}

type Dispatcher struct {
	orkerCmd    []string
	workers     map[string]*Worker
	lastCleanup int64
}

func NewDispatcher() *Dispatcher {
	initSigHandler()
	m := map[string]*Worker{}
	return &Dispatcher{workers: m}
}

func (d *Dispatcher) CleanupAll() {
	log.Println("Finalize!!!!!!!!")
	for _, v := range d.workers {
		if err := DestoryWorker(v); err != nil {
			log.Fatal(err)
		}
	}
}

func (srv *Dispatcher) Handler(w http.ResponseWriter, r *http.Request) {
	//w.Write([]byte("Hello..."))
	targetStr := r.URL.Query().Get("target")
	fmt.Println("target:", targetStr)

	parsedUrl, err := url.Parse("unix://" + targetStr)
	if err != nil {
		log.Fatal(err)
	}
	host, port, _ := net.SplitHostPort(parsedUrl.Host)
	log.Print("host:", host, " port:", port)
	target := Target{host: host, port: port}

	worker, ok := srv.workers[target.id()]
	if !ok {
		log.Print("create new worker!")
		//worker, err := CreateWorker([]string{"sleep", "45"})
		worker, err := CreateWorker(&target)
		if err != nil {
			log.Print("err!!!", err)
			return
		}
		srv.workers[target.id()] = worker
	}
	log.Print("let's request")
	if err := worker.Request(&w, r); err != nil {
		return
	}
	//srv.cleanup()
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

func (srv *Dispatcher) cleanup() {

}
