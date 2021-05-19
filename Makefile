CMD := superexporter
CMDDIR := ./cmd
PKGDIR := ./pkg

.PHONY: ALL
ALL: $(CMD)
$(CMD): $(CMDDIR)/**/*.go $(PKGDIR)/**/*.go
	env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $@ $(CMDDIR)/$@/

clean:
	rm -f $(CMD)
