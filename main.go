package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/Singert/xjtu_cnlab/app"
	"github.com/Singert/xjtu_cnlab/core"
	"github.com/Singert/xjtu_cnlab/core/config"
	"github.com/Singert/xjtu_cnlab/core/server"
	"github.com/Singert/xjtu_cnlab/core/talklog"
)

func main() {

	var httpServer server.ServerInterface

	config.Cfg.StartTime = time.Now()
	gid := talklog.GID()
	// 初始化配置
	config.InitConfig()
	logConfig := &talklog.LogConfig{
		LogToFile: config.Cfg.Logger.LogToFile,
		FilePath:  config.Cfg.Logger.FilePath,
		WithTime:  config.Cfg.Logger.WithTime,
	}
	talklog.InitLogConfig(logConfig)

	// 定义命令行参数（默认值来自配置）
	directory := flag.String("d", config.Cfg.Server.Workdir, "服务目录")
	protocol := flag.String("p", config.Cfg.Server.Proto, "HTTP协议版本")
	ipv4 := flag.String("a", config.Cfg.Server.IPv4, "IPv4地址")
	ipv6 := flag.String("b", config.Cfg.Server.IPv6, "IPv6地址")
	isDualStack := flag.Bool("D", config.Cfg.Server.IsDualStack, "启用双栈支持")
	isCgi := flag.Bool("c", config.Cfg.Server.IsCgi, "启用CGI支持")

	// 支持长参数名
	flag.StringVar(directory, "directory", *directory, "服务目录")
	flag.StringVar(protocol, "protocol", *protocol, "HTTP协议版本")
	flag.StringVar(ipv4, "ipv4", *ipv4, "IPv4地址")
	flag.StringVar(ipv6, "ipv6", *ipv6, "IPv6地址")
	flag.BoolVar(isDualStack, "dualstack", *isDualStack, "启用双栈支持")
	flag.BoolVar(isCgi, "cgi", *isCgi, "启用CGI支持")

	// 解析命令行参数
	flag.Parse()

	// 合并配置
	if *protocol != "" {
		config.Cfg.Server.Proto = *protocol
	}
	talklog.Boot(gid, "服务器版本: %s", config.Cfg.Server.Proto)

	if *directory != "" {
		config.Cfg.Server.Workdir = *directory
	}
	talklog.Boot(gid, "提供目录: %s", config.Cfg.Server.Workdir)

	if *ipv4 != "" {
		config.Cfg.Server.IPv4 = *ipv4
	}
	if *ipv6 != "" {
		config.Cfg.Server.IPv6 = *ipv6
	}
	if *isDualStack {

		config.Cfg.Server.IsDualStack = *isDualStack
	}
	talklog.Boot(gid, "双栈支持: %t", config.Cfg.Server.IsDualStack)

	if *isCgi {
		config.Cfg.Server.IsCgi = *isCgi
	}
	talklog.Boot(gid, "CGI支持: %t", config.Cfg.Server.IsCgi)

	// 获取端口参数
	args := flag.Args()
	if len(args) > 0 {
		p, err := strconv.Atoi(args[0])
		if err == nil && p > 0 && p < 65536 {
			config.Cfg.Server.Port = p
		}
	}
	talklog.Boot(gid, "监听端口: %d", config.Cfg.Server.Port)

	// 启动服务器
	if config.Cfg.Server.IsDualStack {

		// 启动双栈服务器
		srv, err := server.StartDualStackServer()
		if err != nil {
			talklog.Boot(gid, "启动双栈服务器失败: %v", err)
			fmt.Fprintf(os.Stderr, "启动双栈服务器失败: %v\n", err)
			os.Exit(1)
		}
		httpServer = srv
	} else {

		// 启动服务器
		srv, err := server.StartServer()
		if err != nil {
			talklog.Boot(gid, "启动服务器失败: %v", err)
			fmt.Fprintf(os.Stderr, "启动服务器失败: %v\n", err)
			os.Exit(1)
		}
		httpServer = srv
	}

	//注册路由
	app.RegisterAppRoutes(httpServer.GetRouter())
	//注册完成日志
	talklog.Boot(gid, "路由注册完成")
	go func() {
		err := core.Serve(httpServer)
		if err != nil {
			talklog.Boot(gid, "服务器启动失败: %v", err)
			fmt.Fprintf(os.Stderr, "服务器启动失败: %v\n", err)
			os.Exit(1)
		}
	}()

	// 等待服务器关闭
	// 捕获系统信号 (Ctrl+C / kill)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// 阻塞直到收到信号
	sig := <-quit
	talklog.Boot(gid, "收到信号 %s，正在关机...", sig)

	// 调用服务器关机
	if err := httpServer.Shutdown(); err != nil {
		talklog.Boot(gid, "服务器关机失败: %v", err)
	} else {
		talklog.Boot(gid, "服务器关机完成")
	}

}
