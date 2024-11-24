FROM golang:1.23.3 AS building

COPY . /building
WORKDIR /building

RUN make cfddns

FROM alpine:3

RUN apk add --no-cache tzdata
# 创建文件夹
RUN mkdir -p /usr/bin/cfddns
COPY --from=building /building/bin/cfddns /usr/bin/cfddns/cfddns

ENTRYPOINT ["/usr/bin/cfddns/cfddns"]
# CMD ["-c", "/etc/cfddns/conf.toml"]