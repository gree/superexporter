package superexporter

import (
	"testing"
)

func Test_buildCmdStr(t *testing.T) {
	type args struct {
		cmdTpl string
		la     string
		t      *Target
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			"Expand template correctly",
			args{
				cmdTpl: "somebin --opts-listenaddr {{.ListenAddr}} --opts-target {{.Target.Host}}:{{.Target.Port}}",
				la:     "127.0.0.1:12345",
				t:      &Target{Host: "192.168.10.10", Port: "11211"},
			},
			"somebin --opts-listenaddr 127.0.0.1:12345 --opts-target 192.168.10.10:11211",
			false,
		},
		{
			"Expand failured with unexpected placeholders",
			args{
				cmdTpl: "somebin --opts-listenaddr {{.FooBarAddr}}",
				la:     "127.0.0.1:12345",
				t:      &Target{Host: "192.168.10.10", Port: "11211"},
			},
			"",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildCmdStr(tt.args.cmdTpl, tt.args.la, tt.args.t)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildCmdStr() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("buildCmdStr() = %v, want %v", got, tt.want)
			}
		})
	}
}
