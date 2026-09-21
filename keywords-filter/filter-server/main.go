package main

import (
	"context"
	"flag"
	"fmt"
	"keywords-filter/filter-server/server"
	mrpc_filter "keywords-filter/mrpc_generated/filter"
	"keywords-filter/pkg/config"
	"keywords-filter/pkg/filter"
	"keywords-filter/pkg/log"

	mrpc "github.com/yansss626/mrpc/runtime"
)

var (
	configFile = flag.String("config", "dev.config.yaml", "")
	dictFile   = flag.String("dict", "dict.txt", "")
	formatDict = flag.Bool("format", false, "")
)

func main() {
	flag.Parse()
	if *formatDict {
		filter.OverwriteDict(*dictFile)
		return
	}

	//初始化配置文件
	config.InitConfig(*configFile)
	cnf := config.GetConfig()
	//初始化日志
	log.SetLevel(cnf.Log.Level)
	log.SetOutput(log.GetRotateWriter(cnf.Log.LogPath))
	log.SetPrintCaller(true)
	//初始话filter
	filter.InitFilter(*dictFile)

	registry := mrpc.NewRegistry()
	service := server.NewFilterService(filter.GetFilter())
	err := mrpc_filter.RegisterFilterService(registry, service)
	if err != nil {
		log.Fatal(err)
	}
	server := mrpc.NewServer(registry)
	if err = server.Listen(context.Background(), fmt.Sprintf("%s:%d", cnf.Server.IP, cnf.Server.Port)); err != nil {
		log.Fatal(err)
	}
}
