package handler

import (
	"net"
)

// 这个文件作为handler包的入口点，提供了创建各种HTTP请求处理器的工厂函数
// 具体实现已拆分到以下文件中：
// - status.go: HTTP状态码和状态消息定义
// - base_handler.go: 基础HTTP请求处理器实现
// - simple_handler.go: 简单HTTP请求处理器实现
// - cgi_handler.go: CGI HTTP请求处理器实现

// NewHTTPRequestHandler 根据配置创建合适的HTTP请求处理器
// 如果启用了CGI，则返回CGIHTTPRequestHandler
// 否则返回SimpleHTTPRequestHandler
func NewHTTPRequestHandler(conn net.Conn, enableCGI bool) interface{} {
	if enableCGI {
		return NewCGIHTTPRequestHandler(conn)
	}
	return NewSimpleHTTPRequestHandler(conn)
}
