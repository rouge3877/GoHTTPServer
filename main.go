package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	globalconfig "github.com/Singert/xjtu_cnlab/core/global_config"
	"github.com/Singert/xjtu_cnlab/core/server"
	"github.com/Singert/xjtu_cnlab/core/talklog"
)

func main() {
	globalconfig.GlobalConfig.StartTime = time.Now()
	gid := talklog.GID()
	// 初始化配置
	globalconfig.InitConfig()
	logConfig := &talklog.LogConfig{
		LogToFile: globalconfig.GlobalConfig.Logger.LogToFile,
		FilePath:  globalconfig.GlobalConfig.Logger.FilePath,
		WithTime:  globalconfig.GlobalConfig.Logger.WithTime,
	}
	talklog.InitLogConfig(logConfig)

	// 定义命令行参数（默认值来自配置）
	directory := flag.String("d", globalconfig.GlobalConfig.Server.Workdir, "服务目录")
	protocol := flag.String("p", globalconfig.GlobalConfig.Server.Proto, "HTTP协议版本")
	ipv4 := flag.String("a", globalconfig.GlobalConfig.Server.IPv4, "IPv4地址")
	ipv6 := flag.String("b", globalconfig.GlobalConfig.Server.IPv6, "IPv6地址")
	isDualStack := flag.Bool("D", globalconfig.GlobalConfig.Server.IsDualStack, "启用双栈支持")
	isCgi := flag.Bool("c", globalconfig.GlobalConfig.Server.IsCgi, "启用CGI支持")

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
		globalconfig.GlobalConfig.Server.Proto = *protocol
	}
	talklog.Boot(gid, "服务器版本: %s", globalconfig.GlobalConfig.Server.Proto)

	if *directory != "" {
		globalconfig.GlobalConfig.Server.Workdir = *directory
	}
	talklog.Boot(gid, "提供目录: %s", globalconfig.GlobalConfig.Server.Workdir)

	if *ipv4 != "" {
		globalconfig.GlobalConfig.Server.IPv4 = *ipv4
	}
	if *ipv6 != "" {
		globalconfig.GlobalConfig.Server.IPv6 = *ipv6
	}
	if *isDualStack {

		globalconfig.GlobalConfig.Server.IsDualStack = *isDualStack
	}
	talklog.Boot(gid, "双栈支持: %t", globalconfig.GlobalConfig.Server.IsDualStack)

	if *isCgi {
		globalconfig.GlobalConfig.Server.IsCgi = *isCgi
	}
	talklog.Boot(gid, "CGI支持: %t", globalconfig.GlobalConfig.Server.IsCgi)

	// 获取端口参数
	args := flag.Args()
	if len(args) > 0 {
		p, err := strconv.Atoi(args[0])
		if err == nil && p > 0 && p < 65536 {
			globalconfig.GlobalConfig.Server.Port = p
		}
	}
	talklog.Boot(gid, "监听端口: %d", globalconfig.GlobalConfig.Server.Port)

	// 启动服务器
	if globalconfig.GlobalConfig.Server.IsDualStack {

		// 启动双栈服务器
		err := server.StartDualStackServer()
		if err != nil {
			talklog.Boot(gid, "启动双栈服务器失败: %v", err)
			fmt.Fprintf(os.Stderr, "启动双栈服务器失败: %v\n", err)
			os.Exit(1)
		}
	} else {

		// 启动服务器
		err := server.StartServer()
		if err != nil {
			talklog.Boot(gid, "启动服务器失败: %v", err)
			fmt.Fprintf(os.Stderr, "启动服务器失败: %v\n", err)
			os.Exit(1)
		}
	}
}
