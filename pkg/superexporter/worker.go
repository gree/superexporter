package superexporter

import (
	"net/http"
)

type WorkerInterface interface {
	Destroy() error
	Request(writerRef *http.ResponseWriter, r *http.Request) error
}
