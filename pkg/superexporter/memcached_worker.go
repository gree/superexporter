package superexporter

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"text/template"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

type MemcachedWorker struct {
	target *Target
	logger log.Logger
	client *http.Client
	st     MemcachedWorkerStatus
}

type MemcachedWorkerStatus struct {
	pid          int
	childAddress *url.URL
}

var (
	memcachedExporterBin     = os.Getenv("MEMCACHED_EXPORTER_BIN")
	memcachedExporterOptions = os.Getenv("MEMCACHED_EXPORTER_OPTIONS")
)

func init() {
	if memcachedExporterBin == "" {
		memcachedExporterBin = "memcached_exporter"
	}
	fmt.Println("memcached exporter bin: ", memcachedExporterBin)
	if memcachedExporterOptions == "" {
		memcachedExporterOptions = "--web.listen-address {{.ListenAddr}} --memcached.address {{.Target.Host}}:{{.Target.Port}} --web.telemetry-path /metrics"
	}
	fmt.Println("memcached exporter options: ", memcachedExporterOptions)
}

func CreateMemcachedWorker(t *Target, logger log.Logger) (*MemcachedWorker, error) {
	w := &MemcachedWorker{target: t, client: httpClient(), logger: logger}
	err := w.spawn(memcachedExporterBin + " " + memcachedExporterOptions)
	if err != nil {
		return nil, err
	}
	time.Sleep(time.Second * 1) // wait for spawned exporter ready
	return w, nil
}

func (w *MemcachedWorker) Destroy() error {
	return syscall.Kill(w.st.pid, syscall.SIGTERM)
}

func (w *MemcachedWorker) Request(writerRef *http.ResponseWriter, _ *http.Request) error {
	level.Debug(w.logger).Log("msg", "start request")
	response, err := w.client.Get(w.st.childAddress.String() + "/metrics")
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

func buildCmdStr(cmdTpl string, la string, t *Target) (string, error) {
	tmpl, err := template.New("cmdTpl").Parse(cmdTpl)
	if err != nil {
		return "", err
	}
	buf := new(bytes.Buffer)
	err = tmpl.Execute(buf, struct {
		ListenAddr string
		Target     *Target
	}{la, t})
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (w *MemcachedWorker) spawn(cmdTpl string) error {
	localPort, err := probablyAvailableLocalPort()
	if err != nil {
		return err
	}
	u, err := url.Parse("http://localhost:" + localPort)
	if err != nil {
		return err
	}
	level.Debug(w.logger).Log("msg", "cmdTpl: "+cmdTpl)
	cmdStr, err := buildCmdStr(cmdTpl, u.Host, w.target)
	if err != nil {
		return err
	}
	level.Debug(w.logger).Log("msg", "cmdStr: "+cmdStr)

	commands := strings.Split(cmdStr, " ")
	cmd := exec.CommandContext(context.TODO(), commands[0], commands[1:]...)
	if err := cmd.Start(); err != nil {
		return err
	}
	w.st.childAddress = u
	w.st.pid = cmd.Process.Pid
	level.Info(w.logger).Log("msg", fmt.Sprintf("exporter process for %s is running with PID:%d", w.target.id(), w.st.pid))

	return nil
}

func probablyAvailableLocalPort() (string, error) {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return "", err
	}
	addr := l.Addr().String()
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", err
	}
	defer l.Close()
	return port, nil
}

func httpClient() *http.Client {
	return &http.Client{}
}
