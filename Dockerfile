FROM quay.io/projectquay/golang:1.22 AS builder

WORKDIR /app 
COPY . .
ENV CGO_ENABLED=0
RUN go build -ldflags="-w -s" -v -o /app/fastnetmon_exporter . .


FROM scratch
WORKDIR /
COPY --from=builder /app/fastnetmon_exporter /fastnetmon_exporter
EXPOSE 9898
USER 65534:65534
ENTRYPOINT [ "/fastnetmon_exporter" ]