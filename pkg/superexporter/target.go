package superexporter

type Target struct {
	Host string
	Port string
	kind string
}

func (t *Target) id() string {
	return t.Host + "-" + t.Port
}
