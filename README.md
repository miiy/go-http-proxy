# go-http-proxy

A simple reverse proxy written in Go.

## Features

- proxy auth
- local proxy
- log

## docker

build image

```bash
sudo docker build . -t registry.cn-hangzhou.aliyuncs.com/test/go-http-proxy:v0.1.0
```

run

```bash
sudo docker run --name go-http-proxy \
  --add-host=host.docker.internal:host-gateway \
  --net frontend -d registry.cn-hangzhou.aliyuncs.com/test/go-http-proxy:v0.1.0 /app/app -addr=:8080 -local-proxy=http://host.docker.internal:7890
```