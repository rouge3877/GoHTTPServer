package handler

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/Singert/xjtu_cnlab/core/config"
	"github.com/Singert/xjtu_cnlab/core/server"
	"github.com/Singert/xjtu_cnlab/core/talklog"
)

// CGIHTTPRequestHandler 实现CGI HTTP请求处理器
type CGIHTTPRequestHandler struct {
	*SimpleHTTPRequestHandler
	CGIDirectories []string  // CGI脚本目录列表
	cgiInfo        [2]string // 存储匹配的 (dir, rest)
}

// NewCGIHTTPRequestHandler 创建一个新的CGI HTTP请求处理器
func NewCGIHTTPRequestHandler(server *server.HTTPServer, conn net.Conn) *CGIHTTPRequestHandler {
	handler := &CGIHTTPRequestHandler{
		SimpleHTTPRequestHandler: NewSimpleHTTPRequestHandler(server, conn),
		CGIDirectories:           []string{"/cgi-bin", "/htbin", "/cgi", "/api", "app"},
	}
	handler.ProcessMethod = handler

	return handler
}

// Utilities for CGIHTTPRequestHandler
func _url_collapse_path(path string) string {
	/*
	   Given a URL path, remove extra '/'s and '.' path elements and collapse
	   any '..' references and returns a collapsed path.

	   Implements something akin to RFC-2396 5.2 step 6 to parse relative paths.
	   The utility of this function is limited to is_cgi method and helps
	   preventing some security attacks.

	   Returns: The reconstituted URL, which will always start with a '/'.

	   Raises: IndexError if too many '..' occur within the path.
	*/

	// 去除查询参数
	var raw, query string
	if idx := strings.Index(path, "?"); idx != -1 {
		raw = path[:idx]
		query = path[idx+1:]
	} else {
		raw = path
	}
	// 解码 URL 编码
	decoded, _ := url.PathUnescape(raw)

	// 拆分为各段
	parts := strings.Split(decoded, "/")
	head := make([]string, 0, len(parts))
	// 处理中间段（除最后一段外）
	for _, part := range parts[:len(parts)-1] {
		switch part {
		case "", ".":
			// skip
		case "..":
			if len(head) > 0 {
				head = head[:len(head)-1]
			} else {
				panic("too many .. in path")
			}
		default:
			head = append(head, part)
		}
	}
	// 处理尾段
	tail := ""
	if len(parts) > 0 {
		tail = parts[len(parts)-1]
		switch tail {
		case "..":
			if len(head) > 0 {
				head = head[:len(head)-1]
			} else {
				panic("too many .. in path")
			}
			tail = ""
		case ".":
			tail = ""
		}
	}
	// 如果有 query，附加回去
	if query != "" {
		if tail != "" {
			tail = tail + "?" + query
		} else {
			tail = "?" + query
		}
	}
	// 重组路径
	prefix := "/" + strings.Join(head, "/")
	return strings.Join([]string{prefix, tail}, "/")
}

// IsCGIScript 检查路径是否为CGI脚本
func (h *CGIHTTPRequestHandler) IsCGIScript() bool {
	collapsed := _url_collapse_path(h.Path)
	// 从第1位开始查找下一个 '/'
	idx := strings.Index(collapsed[1:], "/")
	if idx >= 0 {
		idx++ // 调整为在 collapsed 中的真实索引
	}
	// 向后继续查找，直到 dir 部分匹配 CGI 目录
	for idx > 0 && !contains(h.CGIDirectories, collapsed[:idx]) {
		next := strings.Index(collapsed[idx+1:], "/")
		if next < 0 {
			idx = -1
			break
		}
		idx += next + 1
	}
	if idx > 0 && contains(h.CGIDirectories, collapsed[:idx]) {
		h.cgiInfo[0] = collapsed[:idx]
		h.cgiInfo[1] = collapsed[idx+1:]
		return true
	}
	return false
}

// Helper: 判断 target 是否在列表中
func contains(list []string, target string) bool {
	for _, v := range list {
		if v == target {
			return true
		}
	}
	return false
}

// DoPOST 处理POST请求
func (h *CGIHTTPRequestHandler) DoPOST() {
	if h.IsCGIScript() {
		h.RunCGI()
	} else {
		h.SimpleHTTPRequestHandler.DoPOST()
	}
}

// sendCGIHeaders 发送CGI响应头
func (h *CGIHTTPRequestHandler) SendHead() (*os.File, error) {
	// fmt.Println("SendHead in CGI!!!")
	if h.IsCGIScript() {
		h.RunCGI()
		return nil, nil
	} else {
		return h.SimpleHTTPRequestHandler.SendHead()
	}
}

// RunCGI 执行CGI脚本
func (h *CGIHTTPRequestHandler) RunCGI() {
	gid := talklog.GID()
	talklog.SetPrefix(gid, "CGI")
	dir := h.cgiInfo[0]
	rest := h.cgiInfo[1]
	scriptPath := path.Join(dir, rest)

	// 查找最长的有效目录路径
	for {
		i := strings.Index(scriptPath[len(dir)+1:], "/")
		if i < 0 {
			break
		}
		i += len(dir) + 1
		nextDir := scriptPath[:i]
		translated := h.TranslatePath(nextDir)
		if fi, err := os.Stat(translated); err == nil && fi.IsDir() {
			dir = nextDir
			rest = scriptPath[i+1:]
		} else {
			break
		}
	}
	talklog.Info(gid, "Ready to run CGI script: %s", scriptPath)
	// 解析查询字符串
	rest, query, _ := strings.Cut(rest, "?")

	// 分割脚本名和PATH_INFO
	i := strings.Index(rest, "/")
	var script, pathInfo string
	if i >= 0 {
		script, pathInfo = rest[:i], rest[i:]
	} else {
		script, pathInfo = rest, ""
	}

	// 构建脚本的完整路径
	scriptFile := h.TranslatePath(path.Join(dir, script))
	talklog.Info(gid, "CGI script path: %s", scriptFile)

	// 检查脚本是否存在且可执行
	fi, err := os.Stat(scriptFile)
	if err != nil {
		h.SendError(NOT_FOUND, fmt.Sprintf("No such CGI script (%s)", script))
		return
	}

	if fi.IsDir() {
		h.SendError(FORBIDDEN, "CGI script is a directory")
		return
	}

	// 构建环境变量
	env := make([]string, 0)

	// 添加基本环境变量
	env = append(env, fmt.Sprintf("SERVER_SOFTWARE=%s/%s", config.GoHTTPServerName(), config.GoHTTPServerVersion()))
	env = append(env, fmt.Sprintf("SERVER_NAME=%s", h.ServerVersion))
	env = append(env, fmt.Sprintf("GATEWAY_INTERFACE=CGI/1.1"))
	env = append(env, fmt.Sprintf("SERVER_PROTOCOL=%s", h.RequestVersion))
	env = append(env, fmt.Sprintf("SERVER_PORT=%d", 8000)) // 假设端口为8000
	env = append(env, fmt.Sprintf("REQUEST_METHOD=%s", h.Command))
	env = append(env, fmt.Sprintf("PATH_INFO=%s", pathInfo))
	env = append(env, fmt.Sprintf("PATH_TRANSLATED=%s", h.TranslatePath(path.Join(dir, pathInfo))))
	env = append(env, fmt.Sprintf("SCRIPT_NAME=%s", path.Join(dir, script)))

	if query != "" {
		env = append(env, fmt.Sprintf("QUERY_STRING=%s", query))
	}

	// 添加HTTP头作为环境变量
	for k, v := range h.Headers {
		k = strings.ReplaceAll(strings.ToUpper(k), "-", "_")
		env = append(env, fmt.Sprintf("HTTP_%s=%s", k, v))
	}

	// 添加REMOTE_ADDR
	env = append(env, fmt.Sprintf("REMOTE_ADDR=%s", h.ClientAddress))

	// 创建命令
	cmd := exec.Command(scriptFile)
	cmd.Env = env
	cmd.Stdin = h.RFile
	cmd.Stdout = h.WFile
	cmd.Stderr = os.Stderr

	// 执行脚本
	err = cmd.Run()
	if err != nil {
		h.SendError(INTERNAL_SERVER_ERROR, fmt.Sprintf("CGI script execution failed: %v", err))
	}
}
