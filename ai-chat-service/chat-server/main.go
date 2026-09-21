package main

import (
	metrics_bus "ai-chat-service/chat-server/metrics-bus"
	"ai-chat-service/chat-server/server"
	"ai-chat-service/mrpc_generated/chat"
	"ai-chat-service/pkg/config"
	"ai-chat-service/pkg/log"
	"context"
	"flag"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	mrpc "github.com/yansss626/go-mrpc/runtime"
)

const hnswIndexPath = "./chat-server/chat-round/index-algorithm/data/hnsw.snapshot"

var (
	configFile = flag.String("config", "dev.config.yaml", "")
)

func main() {
	flag.Parse()
	registry := prometheus.NewRegistry()
	registry.MustRegister(collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	busMetrics := metrics_bus.NewBusMetrics(registry)

	http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	go http.ListenAndServe(":8080", nil)

	//初始化配置文件
	config.InitConfig(*configFile)
	cnf := config.GetConfig()
	//初始化日志
	log.SetLevel(cnf.Log.Level)
	log.SetOutput(log.GetRotateWriter(cnf.Log.LogPath))
	log.SetPrintCaller(true)

	logger := log.NewLogger()
	logger.SetLevel(cnf.Log.Level)
	logger.SetOutput(log.GetRotateWriter(cnf.Log.LogPath))
	logger.SetPrintCaller(true)

	mrpcRegistry := mrpc.NewRegistry()
	service, err := server.NewChatService(cnf, logger, busMetrics)
	if err != nil {
		log.Fatal(err)
	}
	err = chat.RegisterChatService(mrpcRegistry, service)
	if err != nil {
		panic(err)
	}

	server := mrpc.NewServer(mrpcRegistry)
	err = server.Listen(context.Background(), fmt.Sprintf("%s:%d", cnf.Server.IP, cnf.Server.Port))
	if err != nil {
		log.Fatal(err)
	}

}
