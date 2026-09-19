package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"reprise/mrpc_generated/semantic"
	"reprise/pkg/config"
	"reprise/pkg/log"
	"reprise/server"
	"syscall"

	mrpc "github.com/yansss/mrpc/runtime"
)

var (
	configFile = flag.String("config", "dev.config.yaml", "配置文件")
)

func main() {
	flag.Parse()

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

	registry := mrpc.NewRegistry()
	service, err := server.NewRepriseService(cnf, logger)
	if err != nil {
		panic(err)
	}
	err = semantic.RegisterSemantic(registry, service)
	if err != nil {
		panic(err)
	}
	server := mrpc.NewServer(registry)
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Listen(context.Background(), fmt.Sprintf("%s:%d", cnf.Server.IP, cnf.Server.Port))
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	select {
	case <-stop:
		// 保存向量索引
		err := service.SaveVectorIndex()
		if err != nil {
			log.Fatal(err)
		}
	case err := <-serverErr:
		if err != nil {
			log.Fatal(err)
		}
	}

}
