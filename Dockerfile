FROM golang:1.23.3 AS building

COPY . /building
WORKDIR /building

RUN make cfddns

FROM alpine:3

RUN apk add --no-cache tzdata

COPY --from=building /building/bin/cfddns /usr/bin/cfddns

ENTRYPOINT ["/usr/bin/cfddns"]
CMD ["-c", "/etc/cfddns/conf.toml"]