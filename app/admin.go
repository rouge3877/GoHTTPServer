package app

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/Singert/xjtu_cnlab/core/config"
	"github.com/Singert/xjtu_cnlab/core/router"
	"github.com/Singert/xjtu_cnlab/core/talklog"
)

// HandleAdminReload 处理远程配置热重载
func HandleAdminReload(ctx *router.Context) {
	conn := ctx.Conn.(net.Conn)
	writer := bufio.NewWriter(conn)

	err := config.ReloadConfig()
	body := ""

	if err != nil {
		body = fmt.Sprintf("配置重载失败: %v", err)
		writer.WriteString("HTTP/1.1 500 Internal Server Error\r\n")
	} else {
		body = "配置已成功热重载"
		writer.WriteString("HTTP/1.1 200 OK\r\n")
	}

	writer.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	writer.WriteString("Content-Length: " + strconv.Itoa(len(body)) + "\r\n")
	writer.WriteString("\r\n")
	writer.WriteString(body)
	writer.Flush()
}

func HandleDownloadLogs(ctx *router.Context) {
	conn := ctx.Conn.(net.Conn)
	logs := talklog.GetRecentLogs()

	// 构造文件名，比如 logs-20250429.txt
	fileName := fmt.Sprintf("logs-%s.txt", time.Now().Format("20060102-150405"))

	// 写HTTP响应头
	fmt.Fprintf(conn, "HTTP/1.1 200 OK\r\n")
	fmt.Fprintf(conn, "Content-Type: application/octet-stream\r\n")
	fmt.Fprintf(conn, "Content-Disposition: attachment; filename=\"%s\"\r\n", fileName)
	fmt.Fprintf(conn, "\r\n")

	// 写日志内容
	for _, line := range logs {
		fmt.Fprintf(conn, "%s\n", line)
	}
}
