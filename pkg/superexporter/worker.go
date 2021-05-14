package superexporter

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"text/template"
	"time"
)

/* worker for memcached_exporter */
type Worker struct {
	pid      int
	cmd      []string
	sockName string
	client   *http.Client
}

var (
	memcachedExporterBin     = os.Getenv("MEMCACHED_EXPORTER_BIN")
	memcachedExporterOptions = os.Getenv("MEMCACHED_EXPORTER_OPTIONS")
)

func init() {
	if memcachedExporterBin == "" {
		memcachedExporterBin = "memcached_exporter"
	}
	log.Print("exporter bin:", memcachedExporterBin)
	if memcachedExporterOptions == "" {
		memcachedExporterOptions = "--web.listen-address unix://{{.SockName}} --memcached.address {{.Target.Host}}:{{.Target.Port}} --web.telemetry-path /metrics"
	}
	//log.Print("exporter options:", memcachedExporterOptions)
}

func CreateWorker(t *Target) (*Worker, error) {
	log.Print("[worker] tgt:", t)
	tmpl, err := template.New("optTmpl").Parse(memcachedExporterOptions)
	if err != nil {
		log.Print("error: memcachedExporterOptions:", memcachedExporterOptions)
		return nil, err
	}
	buf := new(bytes.Buffer)
	sockName := "./_" + t.id() + ".sock"
	err = tmpl.Execute(buf, struct {
		SockName string
		Target   *Target
	}{sockName, t})
	if err != nil {
		return nil, err
	}
	optStr := buf.String()
	log.Print("exporter options:", optStr)

	cmd := append([]string{memcachedExporterBin}, strings.Split(optStr, " ")...)
	w := &Worker{cmd: cmd, sockName: sockName}
	if err := w.spawn(); err != nil {
		return nil, err
	}
	time.Sleep(time.Second * 1)
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
	//return nil

	response, err := w.client.Get("http://unix/metrics")
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

func (w *Worker) initHttpClient() {

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
