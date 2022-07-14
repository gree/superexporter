package superexporter

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/go-kit/log"
)

func TestMemcachedWorker_spawnWithAvailablePort(t *testing.T) {
	type fields struct {
		pid          int
		cmd          []string
		client       *http.Client
		childAddress *url.URL
		logger       log.Logger
	}
	type args struct {
		cmdTpl string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{"Expect echo with address", fields{}, args{cmdTpl: "echo foobar --opt {{.ListenAddr}}"}, false},
		{"Invalid command template string", fields{}, args{cmdTpl: "echo foobar {{.Example}}"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &MemcachedWorker{
				pid:          tt.fields.pid,
				cmd:          tt.fields.cmd,
				client:       tt.fields.client,
				childAddress: tt.fields.childAddress,
				logger:       tt.fields.logger,
			}
			if err := w.spawnWithAvailablePort(tt.args.cmdTpl); (err != nil) != tt.wantErr {
				t.Errorf("MemcachedWorker.spawnWithAvailablePort() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
