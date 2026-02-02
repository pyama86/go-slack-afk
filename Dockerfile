FROM golang:latest as builder
ADD . /opt/afk
WORKDIR /opt/afk/
ENV CGO_ENABLED=0
RUN GOOS=linux make build

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /opt/afk/bin/afk /bin/afk
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
CMD ["/bin/afk"]
