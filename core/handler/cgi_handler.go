package handler

import (
	"bytes"
	"fmt"
	"io"
	"net"
	_ "net/url"
	"os"
	"os/exec"
	_ "path"
	"path/filepath"
	_ "slices"
	"strconv"
	"strings"

	"github.com/Singert/xjtu_cnlab/core/config"
	"github.com/Singert/xjtu_cnlab/core/server"
	"github.com/Singert/xjtu_cnlab/core/talklog"
)

// CGIHTTPRequestHandler 实现CGI HTTP请求处理器
type CGIHTTPRequestHandler struct {
	*SimpleHTTPRequestHandler
	CGIDirectoriesList []string // CGI脚本目录列表
	ExecutablePath     string   // 可执行文件路径
}

// NewCGIHTTPRequestHandler 创建一个新的CGI HTTP请求处理器
func NewCGIHTTPRequestHandler(server *server.HTTPServer, conn net.Conn) *CGIHTTPRequestHandler {
	handler := &CGIHTTPRequestHandler{
		SimpleHTTPRequestHandler: NewSimpleHTTPRequestHandler(server, conn),
		CGIDirectoriesList:       config.Cfg.Server.CGIDirectories,
		ExecutablePath:           "",
	}
	handler.ProcessMethod = handler

	return handler
}

// IsCGIScript 检查路径是否为CGI脚本
func (h *CGIHTTPRequestHandler) IsCGIScript() bool {
	/*
	   Check if the request path is a CGI script.
	   检查请求路径是否为CGI脚本。
	   Returns: True if the path is a CGI script, False otherwise.
	   返回：如果路径是CGI脚本，则为True，否则为False。
	*/

	var isCGIScript bool
	isCGIScript = false

	for _, dir := range h.CGIDirectoriesList {
		// check if there is any '/dir/' in h.Path
		containCheck := "/" + dir + "/"
		if strings.Contains(h.Path, containCheck) {
			isCGIScript = true
			break
		}
	}

	// check if it's a runable file
	if isCGIScript {
		// check if the file is executable
		filePath := filepath.Join(config.Cfg.Server.Workdir, h.Path)
		// check if the file is executable
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			// fmt.Println("Error: ", err)
			return false
		}
		if fileInfo.Mode()&0111 == 0 {
			// fmt.Println("File is not executable: ", filePath)
			return false
		}
		// check if the file is a directory
		if fileInfo.IsDir() {
			// fmt.Println("File is a directory: ", filePath)
			return false
		}

		// set h.ExecutablePath as the file path
		h.ExecutablePath = filePath
	}
	return isCGIScript
}

// DoGET 处理GET请求
func (h *CGIHTTPRequestHandler) DoGET() {
	if h.IsCGIScript() {
		h.RunCGI()
	} else {
		h.SimpleHTTPRequestHandler.DoGET()
	}
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
	talklog.Info(gid, "CGI script request: %s", h.Path)
	talklog.Info(gid, "CGI script executable: %s", h.ExecutablePath)

	// 解析并验证CGI路径
	pathInfo := _getFileDir(h.ExecutablePath)
	scriptFile := h.ExecutablePath

	// 准备CGI环境变量
	env := h.prepareCGIEnvironment(pathInfo)

	// 添加 Content-Length 和 Content-Type 环境变量 (重要 for POST)
	if contentLength, ok := h.Headers["Content-Length"]; ok {
		env = append(env, fmt.Sprintf("CONTENT_LENGTH=%s", contentLength))
	}
	if contentType, ok := h.Headers["Content-Type"]; ok {
		env = append(env, fmt.Sprintf("CONTENT_TYPE=%s", contentType))
	}

	// 执行CGI脚本
	talklog.Info(gid, "Executing CGI script: %s", scriptFile)
	cmd := exec.Command(scriptFile)
	cmd.Env = env

	// --- Start: Handle POST data ---
	var postData bytes.Buffer // Buffer to hold POST data if any
	if h.Command == "POST" {
		contentLengthStr := h.Headers["Content-Length"]
		if contentLengthStr != "" {
			contentLength, err := strconv.ParseInt(contentLengthStr, 10, 64)
			if err != nil {
				talklog.Error(gid, "Invalid Content-Length: %v", err)
				h.SendError(BAD_REQUEST, "Invalid Content-Length header")
				return
			}
			if contentLength > 0 {
				// Read the POST data from the request body (h.RFile)
				// Use io.LimitReader to avoid reading more than specified
				lr := io.LimitReader(h.RFile, contentLength)
				bytesRead, err := io.Copy(&postData, lr)
				if err != nil && err != io.EOF {
					talklog.Error(gid, "Error reading POST data: %v", err)
					h.SendError(INTERNAL_SERVER_ERROR, "Failed to read request body")
					return
				}
				if bytesRead != contentLength {
					talklog.Warn(gid, "POST data read (%d) differs from Content-Length (%d)", bytesRead, contentLength)
					// Decide how to handle this: error or proceed? For now, proceed.
				}
				talklog.Info(gid, "Read %d bytes of POST data", bytesRead)
				// Set the command's standard input to the buffered POST data
				cmd.Stdin = &postData
			}
		} else {
			// Handle POST requests with no Content-Length (e.g., chunked - less common for CGI)
			// For simplicity, we might disallow this or read until EOF, which could be risky.
			// Currently, we'll assume Content-Length is present for CGI POST.
			talklog.Warn(gid, "POST request received without Content-Length")
		}
	}
	// --- End ---

	// 修改: 不直接写入到连接，而是捕获输出
	output, err := cmd.CombinedOutput() // CombinedOutput reads stdout and stderr
	if err != nil {
		// Check if the error contains the output (useful for script errors printed to stderr)
		errMsg := fmt.Sprintf("CGI script execution failed: %v", err)
		if len(output) > 0 {
			errMsg += fmt.Sprintf("\nScript output:\n%s", string(output))
		}
		talklog.Error(gid, errMsg)
		h.SendError(INTERNAL_SERVER_ERROR, fmt.Sprintf("CGI script execution failed: %v", err))
		return
	}

	// 处理CGI脚本输出
	// Find the end of headers (first blank line)
	headerEnd := bytes.Index(output, []byte("\r\n\r\n"))
	if headerEnd == -1 {
		headerEnd = bytes.Index(output, []byte("\n\n")) // Also check for Unix line endings
	}

	var headers []byte
	var body []byte

	if headerEnd != -1 {
		headers = output[:headerEnd]
		// Skip the blank line separator (\r\n\r\n or \n\n)
		bodyStartIndex := headerEnd + 4
		if bytes.Equal(output[headerEnd:headerEnd+2], []byte("\n\n")) {
			bodyStartIndex = headerEnd + 2
		}
		if bodyStartIndex < len(output) {
			body = output[bodyStartIndex:]
		} else {
			body = []byte{}
		}
	} else {
		// Assume entire output is the body, send default headers
		headers = []byte{}
		body = output
	}

	// Send default OK response first (can be overridden by CGI headers)
	h.SendResponse(OK, "")
	contentTypeSent := false

	// Parse and send CGI headers
	headerLines := bytes.Split(headers, []byte("\n"))
	for _, lineBytes := range headerLines {
		line := strings.TrimSpace(string(lineBytes))
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			// Handle specific headers like Status and Content-Type
			if strings.EqualFold(key, "Status") {
				// Example: Status: 404 Not Found
				statusParts := strings.SplitN(value, " ", 2)
				if len(statusParts) >= 1 {
					statusCode, err := strconv.Atoi(statusParts[0])
					if err == nil {
						statusMsg := ""
						if len(statusParts) > 1 {
							statusMsg = statusParts[1]
						}
						// Override the initial OK response
						h.SendResponseOnly(HTTPStatus(statusCode), statusMsg)
						talklog.Info(gid, "CGI Status header: %s", value)
					} else {
						talklog.Warn(gid, "Invalid CGI Status header: %s", value)
					}
				}
			} else {
				h.SendHeader(key, value)
				if strings.EqualFold(key, "Content-Type") {
					contentTypeSent = true
				}
				talklog.Info(gid, "CGI header: %s: %s", key, value)
			}
		} else {
			talklog.Warn(gid, "Malformed CGI header line: %s", line)
		}
	}

	// Send default Content-Type if not provided by CGI
	if !contentTypeSent {
		h.SendHeader("Content-Type", "text/html") // Or a more appropriate default
	}

	// Send Content-Length for the body
	h.SendHeader("Content-Length", strconv.Itoa(len(body)))
	h.EndHeaders() // Send all collected headers

	// Write the body
	if len(body) > 0 {
		h.WFile.Write(body)
	}
	h.WFile.Flush()
	talklog.Info(gid, "CGI script finished successfully")
}

// resolveCGIPath 解析CGI路径并验证脚本是否存在
func _getFileDir(path string) string {
	// 获取文件目录
	dir := path
	if i := strings.LastIndex(path, "/"); i >= 0 {
		dir = path[:i]
	}
	return dir
}

// prepareCGIEnvironment 准备CGI环境变量
func (h *CGIHTTPRequestHandler) prepareCGIEnvironment(pathInfo string) []string {

	// 构建环境变量列表
	env := []string{
		fmt.Sprintf("SERVER_SOFTWARE=%s/%s", config.GoHTTPServerName(), config.GoHTTPServerVersion()),
		fmt.Sprintf("SERVER_NAME=%s", h.ServerVersion),
		fmt.Sprintf("GATEWAY_INTERFACE=%s", "CGI/1.1"),
		fmt.Sprintf("SERVER_PROTOCOL=%s", h.RequestVersion),
		fmt.Sprintf("SERVER_PORT=%d", config.Cfg.Server.Port),
		fmt.Sprintf("REQUEST_METHOD=%s", h.Command),
		fmt.Sprintf("PATH_INFO=%s", pathInfo),
		fmt.Sprintf("SCRIPT_NAME=%s", h.ExecutablePath),
		fmt.Sprintf("REMOTE_ADDR=%s", h.ClientAddress),
	}

	// 添加查询字符串path.Join(dir, script)),
	if h.QueryRaw != "" {
		env = append(env, fmt.Sprintf("QUERY_STRING=%s", h.QueryRaw))
	}

	// 添加HTTP头作为环境变量
	for k, v := range h.Headers {
		k = strings.ReplaceAll(strings.ToUpper(k), "-", "_")
		env = append(env, fmt.Sprintf("HTTP_%s=%s", k, v))
	}

	// 继承系统环境变量
	for _, envVar := range os.Environ() {
		// 避免覆盖已设置的CGI变量
		if !strings.HasPrefix(envVar, "SERVER_") &&
			!strings.HasPrefix(envVar, "GATEWAY_") &&
			!strings.HasPrefix(envVar, "REQUEST_") &&
			!strings.HasPrefix(envVar, "PATH_") &&
			!strings.HasPrefix(envVar, "SCRIPT_") &&
			!strings.HasPrefix(envVar, "REMOTE_") &&
			!strings.HasPrefix(envVar, "QUERY_") &&
			!strings.HasPrefix(envVar, "HTTP_") {
			env = append(env, envVar)
		}
	}

	return env
}
