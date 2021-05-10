package superexporter

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os/exec"
	"syscall"
)

type Worker struct {
	pid      int
	cmd      []string
	sockName string
	client   *http.Client
}

//func CreateWorker(cmd []string) (*Worker, error) {
func CreateWorker(t *Target) (*Worker, error) {
	log.Print("[worker] tgt:", t)
	w := &Worker{cmd: []string{"sleep", "60"}, sockName: t.id()}
	if err := w.spawn(); err != nil {
		return nil, err
	}
	log.Print("worker created")
	return w, nil
}

func DestoryWorker(w *Worker) error {
	return w.Destroy()
}

func (w *Worker) spawn() error {
	if w.client == nil {
		w.initHttpClient()
	}
	cmd := exec.CommandContext(context.TODO(), w.cmd[0], w.cmd[1:]...)
	if err := cmd.Start(); err != nil {
		return err
	}
	w.pid = cmd.Process.Pid
	return nil
}

func (w *Worker) Destroy() error {
	return syscall.Kill(w.pid, syscall.SIGTERM)
}

func (w *Worker) Release() error {
	return syscall.Kill(w.pid, syscall.SIGTERM)
}

func (w *Worker) Request(writerRef *http.ResponseWriter, r *http.Request) error {
	fmt.Println("request!!")
	//(*writerRef).Write([]byte(`hello world`))
	(*writerRef).Write([]byte(`hello world`))
	return nil

	response, err := w.client.Get("http://unix")
	if err != nil {
		return err
	}
	responseByte, err := httputil.DumpResponse(response, true)
	if err != nil {
		return err
	}
	_, err = (*writerRef).Write(responseByte)
	if err != nil {
		return err
	}
	return nil
}

func (w Worker) initHttpClient() {

	if w.sockName == "" {
		panic("empty socket name")
	}
	w.client = &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", w.sockName)
			},
		},
	}
}
