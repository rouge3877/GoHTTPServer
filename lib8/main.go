package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/user/httpserver/server"
)

func main() {
	// 定义命令行参数
	directory := flag.String("d", ".", "指定要提供服务的目录")
	directoryLong := flag.String("directory", ".", "指定要提供服务的目录")
	protocol := flag.String("p", "HTTP/1.0", "指定HTTP协议版本")
	protocolLong := flag.String("protocol", "HTTP/1.0", "指定HTTP协议版本")
	isDualStack := flag.String("D", "false", "启用双栈支持")
	// isCgi := flag.String("c", "false", "启用CGI支持")
	// isCgiLong := flag.String("cgi", "false", "启用CGI支持")
	// isDualStackLong := flag.String("dualstack", "false", "启用双栈支持")

	// 解析命令行参数
	flag.Parse()

	// 确定要使用的目录
	dir := *directory
	if *directoryLong != "." {
		dir = *directoryLong
	}

	// 确定要使用的协议版本
	proto := *protocol
	if *protocolLong != "HTTP/1.0" {
		proto = *protocolLong
	}

	// 获取端口参数
	port := 8000
	args := flag.Args()
	if len(args) > 0 {
		p, err := strconv.Atoi(args[0])
		if err == nil && p > 0 && p < 65536 {
			port = p
		}
	}

	// 打印服务器信息
	fmt.Printf("服务器版本: %s\n", proto)
	fmt.Printf("提供目录: %s\n", dir)
	fmt.Printf("监听端口: %d\n", port)

	// 解析双栈参数
	isDualStackBool, err := strconv.ParseBool(*isDualStack)
	if err != nil {
		fmt.Fprintf(os.Stderr, "无效的双栈参数: %v\n", err)
		os.Exit(1)
	}
	if isDualStackBool {
		fmt.Println("启用双栈支持")
		// 启动双栈服务器
		err := server.StartDualStackServer(port, dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "启动双栈服务器失败: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Printf("启用IPV4支持\n")
		// 启动服务器
		err := server.StartServer(port, dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "启动服务器失败: %v\n", err)
			os.Exit(1)
		}
	}

}
