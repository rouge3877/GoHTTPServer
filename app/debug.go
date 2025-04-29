package app

import (
	"bufio"
	"encoding/json"
	"net"
	"strconv"
	"strings"

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
