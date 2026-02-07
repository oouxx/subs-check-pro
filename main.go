//go:generate go-winres make --in winres/winres.json --product-version=git-tag --file-version=git-tag --arch=amd64,386,arm64
package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/sinspired/subs-check-pro/app"
	"mosn.io/holmes"
)

// 命令行参数
var (
	flagConfigPath = flag.String("f", "", "配置文件路径")
)

func main() {
	// 启动 pprof 监听服务，通常使用 6060 端口
	go func() {
		log.Println("Starting pprof server on :6060")
		if err := http.ListenAndServe("0.0.0.0:6060", nil); err != nil {
			log.Fatalf("pprof server failed: %v", err)
		}
	}()
	go holmesMonitor()
	// 解析命令行参数
	flag.Parse()

	// 初始化应用
	application := app.New(Version, fmt.Sprintf("%s-%s", Version, CurrentCommit), *flagConfigPath)
	// 版本更新成功通知
	application.InitUpdateInfo()
	slog.Info(fmt.Sprintf("当前版本: %s-%s", Version, CurrentCommit))

	if err := application.Initialize(); err != nil {
		slog.Error(fmt.Sprintf("初始化失败: %v", err))
		os.Exit(1)
	}

	application.Run()
}

func holmesMonitor() {
	h, _ := holmes.New(
		holmes.WithCollectInterval("5s"),
		holmes.WithDumpPath("/app/output"),
		holmes.WithTextDump(),
		holmes.WithMemDump(30, 25, 80, time.Minute),
		holmes.WithCGroup(true), // set cgroup to true when using docker or k8s
	)

	h.EnableMemDump()

	// start the metrics collect and dump loop
	h.Start()
}
