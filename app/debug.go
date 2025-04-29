package app

import (
	"bufio"
	"encoding/json"
	"net"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/Singert/xjtu_cnlab/core/config"
	"github.com/Singert/xjtu_cnlab/core/router"
	"github.com/Singert/xjtu_cnlab/core/talklog"
)

// HandleDebugRoutes 输出当前所有路由
func HandleDebugRoutes(ctx *router.Context) {
	conn := ctx.Conn.(net.Conn)
	writer := bufio.NewWriter(conn)

	routes := ctx.RouterAware.GetRouter().ListRoutes()

	var sb strings.Builder
	sb.WriteString("<html><head><title>路由列表</title></head><body>")
	sb.WriteString("<h1>当前注册路由表</h1><ul>")

	for _, r := range routes {
		sb.WriteString("<li><b>")
		sb.WriteString(r.Method)
		sb.WriteString("</b> ")
		sb.WriteString(r.Pattern)
		sb.WriteString(" [Discrption:")
		sb.WriteString(r.Description)
		sb.WriteString("] </li>")
	}

	sb.WriteString("</ul></body></html>")

	body := sb.String()

	// 写HTTP响应头
	writer.WriteString("HTTP/1.1 200 OK\r\n")
	writer.WriteString("Content-Type: text/html; charset=utf-8\r\n")
	writer.WriteString("Content-Length: " + strconv.Itoa(len(body)) + "\r\n")
	writer.WriteString("\r\n")
	writer.WriteString(body)
	writer.Flush()
}

func HandleDebugRoutesJSON(ctx *router.Context) {
	conn := ctx.Conn.(net.Conn)
	writer := bufio.NewWriter(conn)

	// 构造只包含可序列化字段的新切片
	var routeJSON []struct {
		Method      string `json:"method"`
		Pattern     string `json:"pattern"`
		Discription string `json:"description"`
	}
	for _, r := range ctx.RouterAware.GetRouter().ListRoutes() {
		routeJSON = append(routeJSON, struct {
			Method      string `json:"method"`
			Pattern     string `json:"pattern"`
			Discription string `json:"description"`
		}{
			Method:      r.Method,
			Pattern:     r.Pattern,
			Discription: r.Description,
		})
	}

	bodyBytes, err := json.MarshalIndent(routeJSON, "", "  ")
	if err != nil {
		bodyBytes = []byte(`{"error": "failed to encode routes"}`)
	}

	body := string(bodyBytes)

	// 写HTTP响应头
	writer.WriteString("HTTP/1.1 200 OK\r\n")
	writer.WriteString("Content-Type: application/json; charset=utf-8\r\n")
	writer.WriteString("Content-Length: " + strconv.Itoa(len(body)) + "\r\n")
	writer.WriteString("\r\n")
	writer.WriteString(body)
	writer.Flush()
}

func HandleDebugRoutesSmart(ctx *router.Context) {
	con := ctx.Conn.(net.Conn)
	writer := bufio.NewWriter(con)
	routes := ctx.RouterAware.GetRouter().ListRoutes()

	accept := ctx.Headers["Accept"]

	query := ctx.Query
	var (
		body        string
		contentType string
	)

	// 优先根据 URL 参数判断
	format := ""
	if v, ok := query["content-type"]; ok {
		format = strings.ToLower(v)
	} else {
		// 如果URL参数没有，再根据Accept头推测
		if strings.Contains(accept, "application/json") {
			format = "json"
		} else {
			format = "html"
		}
	}

	if format == "json" {
		// 构造只包含可序列化字段的新切片
		var routeJSON []struct {
			Method      string `json:"method"`
			Pattern     string `json:"pattern"`
			Description string `json:"description"`
		}
		for _, r := range routes {
			routeJSON = append(routeJSON, struct {
				Method      string `json:"method"`
				Pattern     string `json:"pattern"`
				Description string `json:"description"`
			}{
				Method:      r.Method,
				Pattern:     r.Pattern,
				Description: r.Description,
			})
		}

		bodyBytes, err := json.MarshalIndent(routeJSON, "", "  ")
		if err != nil {
			bodyBytes = []byte(`{"error": "failed to encode routes"}`)
		}

		body = string(bodyBytes)
		contentType = "application/json; charset=utf-8"
	} else {
		var sb strings.Builder
		sb.WriteString("<html><head><title>路由列表</title></head><body>")
		sb.WriteString("<h1>当前注册路由表</h1><ul>")

		for _, r := range routes {
			sb.WriteString("<li><b>")
			sb.WriteString(r.Method)
			sb.WriteString("</b> ")
			sb.WriteString(r.Pattern)
			sb.WriteString(" [Description: ")
			sb.WriteString(r.Description)
			sb.WriteString("]</li>")
		}

		sb.WriteString("</ul></body></html>")
		body = sb.String()
		contentType = "text/html; charset=utf-8"
	}

	// 写HTTP响应头
	writer.WriteString("HTTP/1.1 200 OK\r\n")
	writer.WriteString("Content-Type: " + contentType + "\r\n")
	writer.WriteString("Content-Length: " + strconv.Itoa(len(body)) + "\r\n")
	writer.WriteString("\r\n")
	writer.WriteString(body)
	writer.Flush()
}

func HandleLogs(ctx *router.Context) {
	conn := ctx.Conn.(net.Conn)
	writer := bufio.NewWriter(conn)

	logs := talklog.GetRecentLogs()

	// 写响应头（加Connection: close）
	writer.WriteString("HTTP/1.1 200 OK\r\n")
	writer.WriteString("Content-Type: text/html; charset=utf-8\r\n")
	writer.WriteString("Connection: close\r\n\r\n")

	// 写HTML内容
	writer.WriteString("<html><head><title>Server Logs</title></head><body><pre style=\"font-size:13px;\">\n")
	for _, line := range logs {
		writer.WriteString(line + "\n")
	}
	writer.WriteString("</pre></body></html>")

	writer.Flush() // 确保flush
}

// 热更新路由
func HandleUpdateRoute(ctx *router.Context) {
	con := ctx.Conn.(net.Conn)
	writer := bufio.NewWriter(con)

	query := ctx.Query
	var (
		body        string
		contentType string
	)

	var handlerRegistry = map[string]router.HandlerFunc{}

	handlerRegistry["HandleDebugRoutes"] = HandleDebugRoutes
	handlerRegistry["HandleDebugRoutesJSON"] = HandleDebugRoutesJSON
	handlerRegistry["HandleDebugRoutesSmart"] = HandleDebugRoutesSmart
	// 优先根据 URL 参数判断
	method := ""
	if v, ok := query["method"]; ok {
		method = strings.ToUpper(v)
	}

	parttern := ""
	if v, ok := query["pattern"]; ok {
		parttern = v
	}
	if method == "" || parttern == "" {
		body = "method or pattern is empty"
		contentType = "text/plain; charset=utf-8"
		writer.WriteString("HTTP/1.1 400 Bad Request\r\n")
		writer.WriteString("Content-Type: " + contentType + "\r\n")
		writer.WriteString("Content-Length: " + strconv.Itoa(len(body)) + "\r\n")
		writer.WriteString("\r\n")
		writer.WriteString(body)
		writer.Flush()
		return
	}

	description := ""
	if v, ok := query["description"]; ok {
		description = v
	}

	var newHandler router.HandlerFunc
	if v, ok := query["handler"]; ok {
		newHandler = handlerRegistry[v]
	}

	ctx.RouterAware.GetRouter().Update(method, parttern, description, newHandler)
	body = "路由更新成功"
	writer.WriteString("HTTP/1.1 200 OK\r\n")
	writer.WriteString("Content-Type: " + contentType + "\r\n")
	writer.WriteString("Content-Length: " + strconv.Itoa(len(body)) + "\r\n")
	writer.WriteString("\r\n")
	writer.WriteString(body)
	writer.Flush()
}

// /debug/info 返回服务器运行配置
func HandleDebugInfo(ctx *router.Context) {
	conn := ctx.Conn.(net.Conn)
	writer := bufio.NewWriter(conn)
	var (
		body        string
		contentType string
	)
	// 获取服务器配置
	info := map[string]interface{}{
		"enable_tls": config.Cfg.Server.EnableTLS,
		"ipv4":       config.Cfg.Server.IPv4,
		"ipv6":       config.Cfg.Server.IPv6,
		"http_port":  config.Cfg.Server.HTTPPort,
		"https_port": config.Cfg.Server.HTTPSPort,
		"workdir":    config.Cfg.Server.Workdir,
		"is_dual":    config.Cfg.Server.IsDualStack,
	}

	bodyBytes, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		bodyBytes = []byte(`{"error": "failed to encode info"}`)
	}
	body = string(bodyBytes)
	contentType = "application/json; charset=utf-8"
	// 写HTTP响应头
	writer.WriteString("HTTP/1.1 200 OK\r\n")
	writer.WriteString("Content-Type: " + contentType + "\r\n")
	writer.WriteString("Content-Length: " + strconv.Itoa(len(body)) + "\r\n")
	writer.WriteString("\r\n")
	writer.WriteString(body)
	writer.Flush()
	talklog.Info(talklog.GID(), "Debug info requested: %s", body)
}

// /debug/uptime 返回服务器运行时间
func HandleUptime(ctx *router.Context) {
	conn := ctx.Conn.(net.Conn)
	writer := bufio.NewWriter(conn)
	var (
		body        string
		contentType string
	)
	// 获取服务器运行时间
	uptime := time.Since(config.Cfg.StartTime)

	result := map[string]string{
		"uptime": uptime.String(),
		"since":  config.Cfg.StartTime.Format(time.RFC3339),
	}

	bodyBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		bodyBytes = []byte(`{"error": "failed to encode uptime"}`)
	}
	body = string(bodyBytes)
	contentType = "application/json; charset=utf-8"
	// 写HTTP响应头
	writer.WriteString("HTTP/1.1 200 OK\r\n")
	writer.WriteString("Content-Type: " + contentType + "\r\n")
	writer.WriteString("Content-Length: " + strconv.Itoa(len(body)) + "\r\n")
	writer.WriteString("\r\n")
	writer.WriteString(body)
	writer.Flush()
	// 记录日志

	talklog.Info(0, "Uptime requested: %s", uptime)
}

func HandleConnCounts(ctx *router.Context) {
	conn := ctx.Conn.(net.Conn)
	writer := bufio.NewWriter(conn)

	counts := ctx.ConnCount.WgCounter()

	bodyBytes, err := json.MarshalIndent(counts, "", "  ")
	if err != nil {
		bodyBytes = []byte(`{"error": "failed to encode counts"}`)
	}
	body := string(bodyBytes)

	// 写HTTP响应头
	writer.WriteString("HTTP/1.1 200 OK\r\n")
	writer.WriteString("Content-Type: application/json; charset=utf-8\r\n")
	writer.WriteString("Content-Length: " + strconv.Itoa(len(body)) + "\r\n")
	writer.WriteString("\r\n")
	writer.WriteString(body)
	writer.Flush()
}

func HandleGortnCounts(ctx *router.Context) {
	conn := ctx.Conn.(net.Conn)
	writer := bufio.NewWriter(conn)

	counts := runtime.NumGoroutine()

	bodyBytes, err := json.MarshalIndent(counts, "", "  ")
	if err != nil {
		bodyBytes = []byte(`{"error": "failed to encode counts"}`)
	}
	body := string(bodyBytes)

	// 写HTTP响应头
	writer.WriteString("HTTP/1.1 200 OK\r\n")
	writer.WriteString("Content-Type: application/json; charset=utf-8\r\n")
	writer.WriteString("Content-Length: " + strconv.Itoa(len(body)) + "\r\n")
	writer.WriteString("\r\n")
	writer.WriteString(body)
	writer.Flush()
}
