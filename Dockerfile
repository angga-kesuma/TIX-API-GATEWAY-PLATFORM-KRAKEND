#FROM asia-southeast1-docker.pkg.dev/tk-dev-micro/base-image/golang:1.24.5-alpine as builder
FROM golang:1.25-alpine AS builder

ARG GITHUB_TOKEN

RUN apk --no-cache --virtual .build-deps add make gcc musl-dev binutils-gold git

RUN git config --global url."https://${GITHUB_TOKEN}:x-oauth-basic@github.com/".insteadOf "https://github.com/"

RUN go env -w GOPRIVATE=github.com/tiket/*

WORKDIR /plugin-build
WORKDIR /plugin
COPY ./go.mod /plugin
COPY ./go.sum /plugin
COPY ./plugin /plugin
RUN set -ex; \
for f in $(find . -type d -links 2); do \
    cd $f; \
    go build -buildmode=plugin -o /plugin-build; \
    cd /plugin; \
done


#FROM asia-southeast1-docker.pkg.dev/tk-dev-micro/base-image/krakend:2.10.2
FROM krakend:2.13.1

RUN ln -snf /usr/share/zoneinfo/Asia/Jakarta /etc/localtime && echo Asia/Jakarta > /etc/timezone

COPY --from=builder /plugin-build/*.so /opt/krakend/plugins/

#COPY check_plugin.sh /usr/local/bin/check_plugin.sh
#RUN chmod +x /usr/local/bin/check_plugin.sh

ENV FC_ENABLE=1
ENV FC_TEMPLATES="/etc/krakend/templates"
ENV FC_SETTINGS="/etc/krakend/settings"
ENV FC_PLUGIN_DIR="/opt/krakend/plugins"
ENV KRAKEND_PLUGIN_FOLDER="/opt/krakend/plugins"

#ENTRYPOINT ["/bin/sh","-c", "\
#  set -e; \
#  /usr/local/bin/check_plugin.sh; \
#  exec /usr/bin/krakend \"$@\" \
#", "--"]

CMD [ "run", "-c", "/etc/krakend/krakend.json", "-p", "8888" ]
