package app

import (
	"bufio"
	"encoding/json"
	"fmt"
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
	body := strings.Join(logs, "\n")

	writer.WriteString("HTTP/1.1 200 OK\r\n")
	writer.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	writer.WriteString("Content-Length: " + strconv.Itoa(len(body)) + "\r\n\r\n")
	writer.WriteString(body)
	writer.Flush()
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

	counts := config.GetConnCount()

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

func HandleDebugMeta(ctx *router.Context) {
	conn := ctx.Conn.(net.Conn)
	writer := bufio.NewWriter(conn)

	uptime := time.Since(config.Cfg.StartTime)

	html := `<html>
<head>
	<title>服务器状态</title>
	<meta charset="utf-8">
	<style>
		body { font-family: sans-serif; padding: 20px; background-color: #f5f5f5; }
		h1 { color: #333; }
		table { border-collapse: collapse; width: 80%; background-color: #fff; box-shadow: 0 0 10px rgba(0,0,0,0.1); }
		th, td { border: 1px solid #ddd; padding: 10px; text-align: left; }
		th { background-color: #f0f0f0; }
	</style>
</head>
<body>
	<h1>服务器运行信息</h1>
	<table>
		<tr><th>项目</th><th>值</th></tr>
		<tr><td>工作目录</td><td>` + config.Cfg.Server.Workdir + `</td></tr>
		<tr><td>HTTP端口</td><td>` + strconv.Itoa(config.Cfg.Server.HTTPPort) + `</td></tr>
		<tr><td>HTTPS端口</td><td>` + strconv.Itoa(config.Cfg.Server.HTTPSPort) + `</td></tr>
		<tr><td>启用TLS</td><td>` + strconv.FormatBool(config.Cfg.Server.EnableTLS) + `</td></tr>
		<tr><td>IPv4地址</td><td>` + config.Cfg.Server.IPv4 + `</td></tr>
		<tr><td>IPv6地址</td><td>` + config.Cfg.Server.IPv6 + `</td></tr>
		<tr><td>双栈支持</td><td>` + strconv.FormatBool(config.Cfg.Server.IsDualStack) + `</td></tr>
		<tr><td>运行时间</td><td>` + uptime.String() + `</td></tr>
		<tr><td>启动时间</td><td>` + config.Cfg.StartTime.Format(time.RFC3339) + `</td></tr>
		<tr><td>Goroutines数量</td><td>` + strconv.Itoa(runtime.NumGoroutine()) + `</td></tr>
	`

	// // 连接统计
	// for k, v := range ctx.ConnCount.WgCounter() {
	// 	html += `<tr><td>` + k + `连接数</td><td>` + strconv.Itoa(v) + `</td></tr>`
	// }

	html += `
	</table>
</body>
</html>`

	writer.WriteString("HTTP/1.1 200 OK\r\n")
	writer.WriteString("Content-Type: text/html; charset=utf-8\r\n")
	writer.WriteString("Content-Length: " + strconv.Itoa(len(html)) + "\r\n\r\n")
	writer.WriteString(html)
	writer.Flush()

	talklog.Info(talklog.GID(), "Debug meta page requested")
}

func HandleDebugDashboard(ctx *router.Context) {
	conn := ctx.Conn.(net.Conn)
	writer := bufio.NewWriter(conn)

	routes := ctx.RouterAware.GetRouter().ListRoutes()
	uptime := time.Since(config.Cfg.StartTime)

	var sb strings.Builder
	sb.WriteString(`<!DOCTYPE html><html><head><title>服务器调试面板</title><style>
	body { font-family: sans-serif; background: #f9f9f9; padding: 20px; }
	h1, h2 { color: #333; }
	pre { background: #eee; padding: 10px; border-radius: 5px; overflow-x: auto; white-space: pre-wrap; }
	table { border-collapse: collapse; width: 100%; margin-bottom: 20px; }
	th, td { border: 1px solid #ccc; padding: 8px 12px; text-align: left; }
	tr:nth-child(even) { background: #f4f4f4; }
	details { margin-bottom: 20px; }
	button { margin-right: 10px; padding: 6px 12px; }
	input { padding: 6px; width: 100%; margin-bottom: 10px; }
	mark { background-color: yellow; color: black; }
	</style>
	<script>
	let allLogs = "";

	function escapeHtml(text) {
		const map = {
			'&': '&amp;',
			'<': '&lt;',
			'>': '&gt;',
			'"': '&quot;',
			"'": '&#039;',
		};
		return text.replace(/[&<>"']/g, function(m) { return map[m]; });
	}

	function refreshLogs() {
		fetch('/logs')
			.then(resp => resp.text())
			.then(text => {
				allLogs = text;
				filterLogs();
			})
			.catch(() => {
				document.getElementById('logbox').innerHTML = '<span style="color:red;">加载失败</span>';
			});
	}

	function filterLogs() {
		const filter = document.getElementById('logFilter').value.toLowerCase();
		const lines = allLogs.split('\n');
		const filtered = lines.filter(line => line.toLowerCase().includes(filter));
		const highlighted = filtered.map(line => {
			if (filter === "") return escapeHtml(line);
			const regex = new RegExp("(" + filter.replace(/[.*+?^${}()|[\]\\]/g, '\\$&') + ")", "gi");
			return escapeHtml(line).replace(regex, "<mark>$1</mark>");
		});
		document.getElementById('logbox').innerHTML = highlighted.join('<br>');
	}

	setInterval(refreshLogs, 5000);
	window.onload = refreshLogs;
	</script>
	</head><body>
	<h1>🛠️ 服务器调试 Dashboard</h1>
	<div style="margin-bottom:10px;">
		<button onclick="location.href='/admin/download_logs'">⬇️ 下载日志</button>
		<button onclick="location.href='/debug/routes?content-type=json'">⬇️ 下载路由表</button>
	</div>
	`)

	// 路由表
	sb.WriteString(`<details open><summary><h2>🟩 路由表</h2></summary><table><tr><th>Method</th><th>Pattern</th><th>Description</th></tr>`)
	for _, r := range routes {
		sb.WriteString("<tr><td>" + r.Method + "</td><td>" + r.Pattern + "</td><td>" + r.Description + "</td></tr>")
	}
	sb.WriteString("</table></details>")

	// 配置 & 状态
	sb.WriteString(`<details open><summary><h2>🟦 服务器状态</h2></summary><table>`)
	sb.WriteString(fmt.Sprintf(`<tr><td>启用TLS</td><td>%v</td></tr>`, config.Cfg.Server.EnableTLS))
	sb.WriteString(fmt.Sprintf(`<tr><td>IPv4</td><td>%s</td></tr>`, config.Cfg.Server.IPv4))
	sb.WriteString(fmt.Sprintf(`<tr><td>IPv6</td><td>%s</td></tr>`, config.Cfg.Server.IPv6))
	sb.WriteString(fmt.Sprintf(`<tr><td>HTTP端口</td><td>%d</td></tr>`, config.Cfg.Server.HTTPPort))
	sb.WriteString(fmt.Sprintf(`<tr><td>HTTPS端口</td><td>%d</td></tr>`, config.Cfg.Server.HTTPSPort))
	sb.WriteString(fmt.Sprintf(`<tr><td>工作目录</td><td>%s</td></tr>`, config.Cfg.Server.Workdir))
	sb.WriteString(fmt.Sprintf(`<tr><td>是否双栈</td><td>%v</td></tr>`, config.Cfg.Server.IsDualStack))
	sb.WriteString(fmt.Sprintf(`<tr><td>当前运行时间</td><td>%s</td></tr>`, uptime))
	sb.WriteString(fmt.Sprintf(`<tr><td>启动时间</td><td>%s</td></tr>`, config.Cfg.StartTime.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf(`<tr><td>当前Goroutines</td><td>%d</td></tr>`, runtime.NumGoroutine()))
	sb.WriteString(fmt.Sprintf(`<tr><td>当前连接总数</td><td>%d</td></tr>`, config.GetConnCount()))
	sb.WriteString("</table></details>")

	// 日志搜索 + 日志区域
	sb.WriteString(`<details open><summary><h2>🟥 实时日志（可搜索）</h2></summary>
	<input type="text" id="logFilter" placeholder="输入关键词过滤日志..." oninput="filterLogs()">
	<pre id="logbox" style="height: 300px; overflow-y: scroll;"></pre>
	</details>`)

	sb.WriteString("</body></html>")

	html := sb.String()
	writer.WriteString("HTTP/1.1 200 OK\r\n")
	writer.WriteString("Content-Type: text/html; charset=utf-8\r\n")
	writer.WriteString("Content-Length: " + strconv.Itoa(len(html)) + "\r\n\r\n")
	writer.WriteString(html)
	writer.Flush()

	talklog.Info(talklog.GID(), "访问了 /debug/dashboard 调试面板")
}
