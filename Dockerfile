# FROM circleci/golang:1.16 as builder
# WORKDIR /build
# COPY . /build
# RUN make

FROM 712427753857.dkr.ecr.ap-northeast-1.amazonaws.com/gree-monitoring/memcached_exporter:0.9.0-9
# COPY --from=builder /build/superexporter /bin/superexporter
COPY superexporter /bin/superexporter

WORKDIR /app/superexporter
RUN chown -R nobody:nobody /app/superexporter && chmod 755 /app/superexporter
USER       nobody
ENTRYPOINT ["/bin/superexporter"]
EXPOSE     9150
