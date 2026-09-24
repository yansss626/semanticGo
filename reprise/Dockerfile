# 编译阶段
FROM quay.io/0voice/golang:1.25 AS stage0
RUN go env -w GOPROXY=https://proxy.golang.com.cn,https://goproxy.cn,direct
COPY ./ /src/reprise
WORKDIR /src/reprise
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o reprise ./cmd

# 运行阶段
FROM quay.io/0voice/alpine:3.18 AS stage1
WORKDIR /app/
ADD ./dev.config.yaml /app/config.yaml
COPY --from=stage0 /src/reprise/reprise ./
ENTRYPOINT [ "./reprise" ]
CMD [ "--config=config.yaml" ]
