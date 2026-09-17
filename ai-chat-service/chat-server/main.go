package main

import (
	chat_context "ai-chat-service/chat-server/chat-context"
	metrics_bus "ai-chat-service/chat-server/metrics-bus"
	"ai-chat-service/chat-server/server"
	"ai-chat-service/mrpc_generated/chat"
	"ai-chat-service/pkg/config"
	"ai-chat-service/pkg/log"
	keywords_filter "ai-chat-service/services/keywords-filter"
	"context"
	"flag"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	mrpc "github.com/yansss/mrpc/runtime"
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

	// 初始化敏感词/关键词服务连接池
	filterClientPool, err := keywords_filter.InitFilterClientPool(cnf)
	if err != nil {
		log.Fatal(err)
	}

	// 初始化上下文缓存
	contextCache, err := chat_context.NewLidisCache(cnf)
	if err != nil {
		log.Fatal(err)
	}

	mrpcRegistry := mrpc.NewRegistry()
	service := server.NewChatService(cnf, logger, busMetrics, contextCache, filterClientPool)
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
