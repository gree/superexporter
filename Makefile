CMD := superexporter
CMDDIR := ./cmd
PKGDIR := ./pkg

.PHONY: ALL
ALL: $(CMD)
$(CMD): $(CMDDIR)/**/*.go $(PKGDIR)/**/*.go
	go build -o $@ $(CMDDIR)/$@/

clean:
	rm -f $(CMD)
