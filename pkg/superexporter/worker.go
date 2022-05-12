package superexporter

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"text/template"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

/* worker for memcached_exporter */
type Worker struct {
	pid      int
	cmd      []string
	sockName string
	client   *http.Client
	logger   log.Logger
}

var (
	memcachedExporterBin     = os.Getenv("MEMCACHED_EXPORTER_BIN")
	memcachedExporterOptions = os.Getenv("MEMCACHED_EXPORTER_OPTIONS")
)

func init() {
	if memcachedExporterBin == "" {
		memcachedExporterBin = "memcached_exporter"
	}
	fmt.Println("exporter bin:", memcachedExporterBin)
	if memcachedExporterOptions == "" {
		memcachedExporterOptions = "--web.listen-address unix://{{.SockName}} --memcached.address {{.Target.Host}}:{{.Target.Port}} --web.telemetry-path /metrics"
	}
}

func CreateWorker(t *Target, logger log.Logger) (*Worker, error) {
	tmpl, err := template.New("optTmpl").Parse(memcachedExporterOptions)
	if err != nil {
		level.Error(logger).Log("msg", "memcachedExporterOptions parse error", "err", fmt.Sprintf("error: memcachedExporterOptions: %s", memcachedExporterOptions))
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
	level.Info(logger).Log("msg", fmt.Sprintf("exporter options: %s", optStr))

	cmd := append([]string{memcachedExporterBin}, strings.Split(optStr, " ")...)
	w := &Worker{cmd: cmd, sockName: sockName, logger: logger}
	if err := w.spawn(); err != nil {
		return nil, err
	}
	time.Sleep(time.Second * 1)
	level.Info(logger).Log("msg", fmt.Sprintf("worker created: %s", t.id()))
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
	level.Debug(w.logger).Log("msg", "start request")

	response, err := w.client.Get("http://unix/metrics")
	if err != nil {
		return err
	}
	defer response.Body.Close()
	responseBodyByte, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	_, err = (*writerRef).Write(responseBodyByte)
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
