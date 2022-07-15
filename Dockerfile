ARG ARCH="amd64"
ARG OS="linux"

FROM quay.io/prometheus/memcached-exporter:v0.10.0 as bin

FROM quay.io/prometheus/busybox-${OS}-${ARCH}:latest
COPY --from=bin /bin/memcached_exporter /bin/memcached_exporter
COPY superexporter /bin/superexporter

WORKDIR /app/superexporter
RUN chown -R nobody:nobody /app/superexporter && chmod 755 /app/superexporter
USER       nobody
ENTRYPOINT ["/bin/superexporter"]
EXPOSE     9150
