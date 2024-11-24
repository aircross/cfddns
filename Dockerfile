FROM golang:1.23.3 AS building

COPY . /building
WORKDIR /building

RUN make cfddns

FROM alpine:3
# 创建文件夹
RUN mkdir -p /usr/bin/cfddns

WORKDIR /usr/bin/cfddns

RUN apk add --no-cache tzdata
COPY --from=building /building/bin/cfddns /usr/bin/cfddns/cfddns
# 从示例中复制配置文件
COPY --from=building /building/conf.toml.example /usr/bin/cfddns/conf.toml

ENTRYPOINT ["/usr/bin/cfddns/cfddns"]