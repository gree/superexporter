ARG ARCH="amd64"
ARG OS="linux"

FROM 712427753857.dkr.ecr.ap-northeast-1.amazonaws.com/gree-monitoring/memcached_exporter:0.9.0-9 as bin

FROM quay.io/prometheus/busybox-${OS}-${ARCH}:latest
COPY --from=bin /bin/memcached_exporter /bin/memcached_exporter
COPY superexporter /bin/superexporter

WORKDIR /app/superexporter
RUN chown -R nobody:nobody /app/superexporter && chmod 755 /app/superexporter
USER       nobody
ENTRYPOINT ["/bin/superexporter"]
EXPOSE     9150
