ARG GO_VERSION=1.21

FROM golang:${GO_VERSION}-bookworm AS builder

RUN go env -w GO111MODULE=on
RUN go env -w GOPROXY=https://goproxy.cn,direct

WORKDIR /go/build
COPY . .
RUN go build -o /go/build/app .


FROM debian:bookworm

ENV TZ=Asia/Shanghai

RUN sed -i 's/deb.debian.org/mirrors.ustc.edu.cn/g' /etc/apt/sources.list \
    && sed -i 's/security.debian.org/mirrors.ustc.edu.cn/g' /etc/apt/sources.list \
    && apt update \
    && apt install -y ca-certificates curl \
    && rm -rf /var/lib/apt/lists/*

RUN mkdir -p /app
WORKDIR /app
COPY --from=builder /go/build/app /app

EXPOSE 8080

CMD ["./app"]
