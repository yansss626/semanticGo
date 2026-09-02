package main

import (
	chat_round "ai-chat-service/chat-server/chat-round"
	"ai-chat-service/chat-server/data"
	metrics_app "ai-chat-service/chat-server/metrics-app"
	metrics_bus "ai-chat-service/chat-server/metrics-bus"
	"ai-chat-service/chat-server/server"
	vector_data "ai-chat-service/chat-server/vector-data"
	"ai-chat-service/interceptor"
	"ai-chat-service/pkg/config"
	"ai-chat-service/pkg/db/kvstore"
	"ai-chat-service/pkg/db/mysql"
	"ai-chat-service/pkg/db/redis"
	"ai-chat-service/pkg/db/vector"
	"ai-chat-service/pkg/log"
	"ai-chat-service/proto"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

	"net"
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

	// 初始化Mysql
	if cnf.Mysql.Enabled {
		mysql.InitMysql(cnf)
	}
	// 初始化redis
	redis.InitRedisPool(cnf)
	// 初始化向量数据库
	if cnf.VectorDB.Enabled {
		vector.InitDB(cnf)
	}
	// 初始化kvstore
	if cnf.Kvstore.Enabled {
		kvstore.InitKvstorePool(cnf)
	}
	// 初始化向量索引
	if cnf.Kvstore.Enabled {
		err := chat_round.InitialVectorIndex(hnswIndexPath, cnf.Embedding.VectorDimensions)
		if err != nil {
			log.Fatal(err)
		}
	}

	recordsData := data.NewChatRecordsData(mysql.GetDB())

	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", cnf.Server.IP, cnf.Server.Port))
	if err != nil {
		log.Fatal(err)
	}
	s := grpc.NewServer(grpc.UnaryInterceptor(interceptor.UnaryAuthInterceptor), grpc.StreamInterceptor(metrics_app.NewStreamMiddleware(registry).WrapHandler()))
	service := server.NewChatService(recordsData, vector_data.NewChatRecordsData(cnf, vector.GetVdb()), cnf, logger, busMetrics)
	proto.RegisterChatServer(s, service)

	healthCheckSrv := health.NewServer()
	grpc_health_v1.RegisterHealthServer(s, healthCheckSrv)

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- s.Serve(lis)
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		if err != nil {
			log.Fatal(err)
		}
	case <-stop:
		s.GracefulStop()
		err := chat_round.SaveVectorIndex(hnswIndexPath)
		if err != nil {
			log.Error(err)
		}
	}
}
