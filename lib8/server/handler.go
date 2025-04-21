package server

import (
	"bufio"
	"fmt"
	"html"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/textproto"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// HTTPStatus 定义HTTP状态码常量
type HTTPStatus int

// HTTP状态码常量定义
const (
	OK                              HTTPStatus = 200
	CREATED                         HTTPStatus = 201
	ACCEPTED                        HTTPStatus = 202
	NO_CONTENT                      HTTPStatus = 204
	RESET_CONTENT                   HTTPStatus = 205
	MOVED_PERMANENTLY               HTTPStatus = 301
	FOUND                           HTTPStatus = 302
	SEE_OTHER                       HTTPStatus = 303
	NOT_MODIFIED                    HTTPStatus = 304
	TEMPORARY_REDIRECT              HTTPStatus = 307
	BAD_REQUEST                     HTTPStatus = 400
	UNAUTHORIZED                    HTTPStatus = 401
	FORBIDDEN                       HTTPStatus = 403
	NOT_FOUND                       HTTPStatus = 404
	METHOD_NOT_ALLOWED              HTTPStatus = 405
	REQUEST_TIMEOUT                 HTTPStatus = 408
	CONFLICT                        HTTPStatus = 409
	GONE                            HTTPStatus = 410
	LENGTH_REQUIRED                 HTTPStatus = 411
	INTERNAL_SERVER_ERROR           HTTPStatus = 500
	NOT_IMPLEMENTED                 HTTPStatus = 501
	BAD_GATEWAY                     HTTPStatus = 502
	SERVICE_UNAVAILABLE             HTTPStatus = 503
	HTTP_VERSION_NOT_SUPPORTED      HTTPStatus = 505
	REQUEST_URI_TOO_LONG            HTTPStatus = 414
	REQUEST_HEADER_FIELDS_TOO_LARGE HTTPStatus = 431
	CONTINUE                        HTTPStatus = 100
)

// 状态码对应的短消息和长消息
var statusMessages = map[HTTPStatus][]string{
	OK:                              {"OK", "Request fulfilled, document follows"},
	CREATED:                         {"Created", "Document created, URL follows"},
	ACCEPTED:                        {"Accepted", "Request accepted, processing continues"},
	NO_CONTENT:                      {"No Content", "Request fulfilled, nothing follows"},
	RESET_CONTENT:                   {"Reset Content", "Clear input form for further input"},
	MOVED_PERMANENTLY:               {"Moved Permanently", "Object moved permanently"},
	FOUND:                           {"Found", "Object moved temporarily"},
	SEE_OTHER:                       {"See Other", "Object moved"},
	NOT_MODIFIED:                    {"Not Modified", "Document has not changed"},
	BAD_REQUEST:                     {"Bad Request", "Bad request syntax or unsupported method"},
	UNAUTHORIZED:                    {"Unauthorized", "No permission"},
	FORBIDDEN:                       {"Forbidden", "Request forbidden"},
	NOT_FOUND:                       {"Not Found", "Nothing matches the given URI"},
	METHOD_NOT_ALLOWED:              {"Method Not Allowed", "Specified method is invalid for this resource"},
	REQUEST_TIMEOUT:                 {"Request Timeout", "Request timed out"},
	CONFLICT:                        {"Conflict", "Request conflict"},
	GONE:                            {"Gone", "URI no longer exists and has been permanently removed"},
	LENGTH_REQUIRED:                 {"Length Required", "Client must specify Content-Length"},
	INTERNAL_SERVER_ERROR:           {"Internal Server Error", "Server got itself in trouble"},
	NOT_IMPLEMENTED:                 {"Not Implemented", "Server does not support this operation"},
	BAD_GATEWAY:                     {"Bad Gateway", "Invalid responses from another server/proxy"},
	SERVICE_UNAVAILABLE:             {"Service Unavailable", "The server cannot process the request due to a high load"},
	HTTP_VERSION_NOT_SUPPORTED:      {"HTTP Version Not Supported", "Cannot fulfill request"},
	REQUEST_URI_TOO_LONG:            {"Request-URI Too Long", "The URI provided was too long for the server to process"},
	REQUEST_HEADER_FIELDS_TOO_LARGE: {"Request Header Fields Too Large", "The server refused this request because the request header fields are too large"},
	CONTINUE:                        {"Continue", "Client should continue with request"},
}

// 默认错误消息模板
const defaultErrorMessageFormat = `<!DOCTYPE HTML>
<html lang="en">
    <head>
        <meta charset="utf-8">
        <title>Error response</title>
    </head>
    <body>
        <h1>Error response</h1>
        <p>Error code: %d</p>
        <p>Message: %s.</p>
        <p>Error code explanation: %d - %s.</p>
    </body>
</html>
`

const defaultErrorContentType = "text/html;charset=utf-8"

type ProcessMethod interface {
	DoGET()
	DoHEAD()
	DoPOST()
	DoPUT()
	DoDELETE()
	DoOPTIONS()
}

// BaseHTTPRequestHandler 实现基本的HTTP请求处理器
type BaseHTTPRequestHandler struct {
	Conn                  net.Conn          // 客户端连接
	Command               string            // 请求命令（GET, POST等）
	Path                  string            // 请求路径
	RequestVersion        string            // 请求HTTP版本
	Headers               map[string]string // 请求头
	RFile                 *bufio.Reader     // 请求读取器
	WFile                 *bufio.Writer     // 响应写入器
	CloseConnection       bool              // 是否关闭连接
	RequestLine           string            // 请求行
	ClientAddress         string            // 客户端地址
	ServerVersion         string            // 服务器版本
	SysVersion            string            // 系统版本
	ErrorMessageFormat    string            // 错误消息格式
	ErrorContentType      string            // 错误内容类型
	ProtocolVersion       string            // 协议版本
	DefaultRequestVersion string            // 默认请求版本
	HeadersBuffer         [][]byte          // 响应头缓冲区

	ProcessMethod ProcessMethod // 处理方法接口
}

// NewBaseHTTPRequestHandler 创建一个新的基本HTTP请求处理器
func NewBaseHTTPRequestHandler(conn net.Conn) *BaseHTTPRequestHandler {

	return &BaseHTTPRequestHandler{
		Conn:                  conn,
		RFile:                 bufio.NewReader(conn),
		WFile:                 bufio.NewWriter(conn),
		CloseConnection:       true,
		ServerVersion:         "GoHTTPServer/0.6",
		SysVersion:            "Go/" + strings.Split(GoVersion(), " ")[0],
		ErrorMessageFormat:    defaultErrorMessageFormat,
		ErrorContentType:      defaultErrorContentType,
		ProtocolVersion:       "HTTP/1.0",
		DefaultRequestVersion: "HTTP/0.9",
		HeadersBuffer:         make([][]byte, 0),
	}
}

// GoVersion 返回Go版本
func GoVersion() string {
	return "1.20"
}

// Handle 处理HTTP请求
func (h *BaseHTTPRequestHandler) Handle() {
	h.CloseConnection = true

	h.HandleOneRequest()
	for !h.CloseConnection {
		h.HandleOneRequest()
	}
}

// HandleOneRequest 处理单个HTTP请求
func (h *BaseHTTPRequestHandler) HandleOneRequest() {
	try := func() {

		if h.RFile == nil {
			fmt.Fprintf(os.Stderr, "Error: RFile is nil\n")
			h.CloseConnection = true
			return
		}

		// 读取请求行
		requestLine, err := h.RFile.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				fmt.Fprintf(os.Stderr, "Error reading request line: %v\n", err)
			}
			h.CloseConnection = true
			return
		}

		// 如果请求行太长，返回错误
		if len(requestLine) > 65536 {
			h.RequestLine = ""
			h.RequestVersion = ""
			h.Command = ""
			h.SendError(REQUEST_URI_TOO_LONG, "")
			return
		}

		// 如果请求行为空，关闭连接
		if len(requestLine) == 0 {
			h.CloseConnection = true
			return
		}

		// 解析请求
		if !h.ParseRequest(requestLine) {
			// 错误已经发送，直接返回
			return
		}
		// 根据请求命令调用相应的处理方法
		mname := "Do" + h.Command
		method := h.GetMethod(mname)
		if method == nil {
			h.SendError(NOT_IMPLEMENTED, fmt.Sprintf("Unsupported method (%s)", h.Command))
			return
		}

		// 调用处理方法
		method()
		// 刷新响应
		h.WFile.Flush()
	}

	// 捕获超时错误
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "Request timed out: %v\n", r)
			h.CloseConnection = true
		}
	}()

	try()
}

// 子类重写 GetMethod
func (h *BaseHTTPRequestHandler) GetMethod(name string) func() {
	// print name
	// fmt.Println(name)
	switch name {
	case "DoGET":
		fmt.Println("DoGET")
		return h.ProcessMethod.DoGET
	case "DoHEAD":
		return h.ProcessMethod.DoHEAD
	case "DoPOST":
		return h.ProcessMethod.DoPOST
	case "DoPUT":
		return h.ProcessMethod.DoPUT
	case "DoDELETE":
		return h.ProcessMethod.DoDELETE
	case "DoOPTIONS":
		return h.ProcessMethod.DoOPTIONS
	}
	return nil
}

// ParseRequest 解析HTTP请求
func (h *BaseHTTPRequestHandler) ParseRequest(requestLine string) bool {
	h.Command = "" // 设置为空，以防解析第一行出错
	h.RequestVersion = h.DefaultRequestVersion
	h.CloseConnection = true

	// 去除请求行末尾的回车换行符
	requestLine = strings.TrimRight(requestLine, "\r\n")
	h.RequestLine = requestLine

	// 分割请求行
	words := strings.Split(requestLine, " ")
	if len(words) == 0 {
		return false
	}

	// 解析HTTP版本
	if len(words) >= 3 {
		version := words[len(words)-1]
		try := func() bool {
			if !strings.HasPrefix(version, "HTTP/") {
				return false
			}

			baseVersionNumber := strings.Split(version, "/")[1]
			versionNumbers := strings.Split(baseVersionNumber, ".")

			// 检查版本号格式
			if len(versionNumbers) != 2 {
				return false
			}

			// 检查版本号是否为数字
			for _, component := range versionNumbers {
				for _, c := range component {
					if c < '0' || c > '9' {
						return false
					}
				}
			}

			// 检查版本号长度是否合理
			for _, component := range versionNumbers {
				if len(component) > 10 {
					return false
				}
			}

			// 解析版本号
			major, _ := strconv.Atoi(versionNumbers[0])
			minor, _ := strconv.Atoi(versionNumbers[1])

			// 根据版本号设置连接关闭标志
			if major*10+minor >= 11 && h.ProtocolVersion >= "HTTP/1.1" {
				h.CloseConnection = false
			}

			// 检查HTTP版本是否支持
			if major >= 2 {
				h.SendError(HTTP_VERSION_NOT_SUPPORTED, fmt.Sprintf("Invalid HTTP version (%s)", baseVersionNumber))
				return false
			}

			h.RequestVersion = version
			return true
		}

		if !try() {
			h.SendError(BAD_REQUEST, fmt.Sprintf("Bad request version (%s)", version))
			return false
		}
	}

	// 检查请求行格式
	if !(2 <= len(words) && len(words) <= 3) {
		h.SendError(BAD_REQUEST, fmt.Sprintf("Bad request syntax (%s)", requestLine))
		return false
	}

	// 解析命令和路径
	command, path := words[0], words[1]
	if len(words) == 2 {
		h.CloseConnection = true
		if command != "GET" {
			h.SendError(BAD_REQUEST, fmt.Sprintf("Bad HTTP/0.9 request type (%s)", command))
			return false
		}
	}

	h.Command, h.Path = command, path

	// 防止开放重定向攻击
	if strings.HasPrefix(h.Path, "//") {
		h.Path = "/" + strings.TrimLeft(h.Path, "/")
	}

	// 解析请求头
	tr := textproto.NewReader(h.RFile)
	headers, err := tr.ReadMIMEHeader()
	if err != nil {
		if err != io.EOF {
			var msg string
			if strings.Contains(err.Error(), "too long") {
				msg = "Line too long"
			} else {
				msg = "Too many headers"
			}
			h.SendError(REQUEST_HEADER_FIELDS_TOO_LARGE, msg, err.Error())
			return false
		}
	}

	// 转换请求头为map
	h.Headers = make(map[string]string)
	for k, v := range headers {
		if len(v) > 0 {
			h.Headers[k] = v[0]
		}
	}

	// 检查Connection头
	connType := h.Headers["Connection"]
	if strings.ToLower(connType) == "close" {
		h.CloseConnection = true
	} else if strings.ToLower(connType) == "keep-alive" && h.ProtocolVersion >= "HTTP/1.1" {
		h.CloseConnection = false
	}

	// 处理Expect头
	expect := h.Headers["Expect"]
	if strings.ToLower(expect) == "100-continue" && h.ProtocolVersion >= "HTTP/1.1" && h.RequestVersion >= "HTTP/1.1" {
		if !h.HandleExpect100() {
			return false
		}
	}

	return true
}

// HandleExpect100 处理Expect: 100-continue头
func (h *BaseHTTPRequestHandler) HandleExpect100() bool {
	h.SendResponseOnly(CONTINUE, "")
	h.EndHeaders()
	return true
}

// SendError 发送错误响应
func (h *BaseHTTPRequestHandler) SendError(code HTTPStatus, message string, args ...string) {
	var shortMsg, longMsg string

	// 获取状态码对应的消息
	if msgs, ok := statusMessages[code]; ok {
		shortMsg, longMsg = msgs[0], msgs[1]
	} else {
		shortMsg, longMsg = "???", "???"
	}

	// 如果没有提供消息，使用默认短消息
	if message == "" {
		message = shortMsg
	}

	// 记录错误
	h.LogError("code %d, message %s", code, message)

	// 发送响应
	h.SendResponse(code, message)
	h.SendHeader("Connection", "close")

	// 某些状态码不需要消息体
	var body []byte
	if code >= 200 && code != NO_CONTENT && code != RESET_CONTENT && code != NOT_MODIFIED {
		// HTML编码以防止跨站脚本攻击
		explain := longMsg
		if len(args) > 0 {
			explain = args[0]
		}

		content := fmt.Sprintf(h.ErrorMessageFormat,
			code,
			html.EscapeString(message),
			code,
			html.EscapeString(explain),
		)

		body = []byte(content)
		h.SendHeader("Content-Type", h.ErrorContentType)
		h.SendHeader("Content-Length", strconv.Itoa(len(body)))
	}

	h.EndHeaders()

	// 发送消息体
	if h.Command != "HEAD" && body != nil {
		h.WFile.Write(body)
	}
}

// LogError 记录错误
func (h *BaseHTTPRequestHandler) LogError(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

// SendResponse 发送响应
func (h *BaseHTTPRequestHandler) SendResponse(code HTTPStatus, message string) {
	h.LogRequest(code, 0)
	h.SendResponseOnly(code, message)
	h.SendHeader("Server", h.VersionString())
	h.SendHeader("Date", h.DateTimeString())
}

// SendResponseOnly 只发送响应行
func (h *BaseHTTPRequestHandler) SendResponseOnly(code HTTPStatus, message string) {
	if h.RequestVersion != "HTTP/0.9" {
		if message == "" {
			if msgs, ok := statusMessages[code]; ok {
				message = msgs[0]
			} else {
				message = ""
			}
		}

		if h.HeadersBuffer == nil {
			h.HeadersBuffer = make([][]byte, 0)
		}

		h.HeadersBuffer = append(h.HeadersBuffer, []byte(fmt.Sprintf("%s %d %s\r\n",
			h.ProtocolVersion, code, message)))
	}
}

// SendHeader 发送HTTP头
func (h *BaseHTTPRequestHandler) SendHeader(keyword, value string) {
	if h.RequestVersion != "HTTP/0.9" {
		if h.HeadersBuffer == nil {
			h.HeadersBuffer = make([][]byte, 0)
		}

		h.HeadersBuffer = append(h.HeadersBuffer, []byte(fmt.Sprintf("%s: %s\r\n", keyword, value)))
	}

	// 处理Connection头
	if strings.ToLower(keyword) == "connection" {
		if strings.ToLower(value) == "close" {
			h.CloseConnection = true
		} else if strings.ToLower(value) == "keep-alive" {
			h.CloseConnection = false
		}
	}
}

// EndHeaders 结束HTTP头部分
func (h *BaseHTTPRequestHandler) EndHeaders() {
	if h.RequestVersion != "HTTP/0.9" {
		h.HeadersBuffer = append(h.HeadersBuffer, []byte("\r\n"))
		h.FlushHeaders()
	}
}

// FlushHeaders 刷新HTTP头
func (h *BaseHTTPRequestHandler) FlushHeaders() {
	if h.HeadersBuffer != nil {
		for _, header := range h.HeadersBuffer {
			h.WFile.Write(header)
		}
		h.HeadersBuffer = make([][]byte, 0)
	}
}

// LogRequest 记录请求
func (h *BaseHTTPRequestHandler) LogRequest(code HTTPStatus, size int) {
	fmt.Printf("%s - - [%s] \"%s\" %d %d\n",
		h.ClientAddress,
		h.LogDate(),
		h.RequestLine,
		code,
		size,
	)
}

// LogDate 返回日志日期格式
func (h *BaseHTTPRequestHandler) LogDate() string {
	now := time.Now()
	return now.Format("02/Jan/2006:15:04:05 -0700")
}

// VersionString 返回服务器版本字符串
func (h *BaseHTTPRequestHandler) VersionString() string {
	return h.ServerVersion + " " + h.SysVersion
}

// DateTimeString 返回HTTP日期时间字符串
func (h *BaseHTTPRequestHandler) DateTimeString() string {
	now := time.Now().UTC()
	return now.Format(time.RFC1123)
}

// DoPOST 处理POST请求
func (h *BaseHTTPRequestHandler) DoPOST() {
	// 默认实现，子类应该重写此方法
	h.SendError(NOT_IMPLEMENTED, "Method not implemented")
}

// DoPUT 处理PUT请求
func (h *BaseHTTPRequestHandler) DoPUT() {
	// 默认实现，子类应该重写此方法
	h.SendError(NOT_IMPLEMENTED, "Method not implemented")
}

// DoDELETE 处理DELETE请求
func (h *BaseHTTPRequestHandler) DoDELETE() {
	// 默认实现，子类应该重写此方法
	h.SendError(NOT_IMPLEMENTED, "Method not implemented")
}

// DoOPTIONS 处理OPTIONS请求
func (h *BaseHTTPRequestHandler) DoOPTIONS() {
	// 默认实现，子类应该重写此方法
	h.SendError(NOT_IMPLEMENTED, "Method not implemented")
}

// SimpleHTTPRequestHandler 实现简单的HTTP请求处理器
type SimpleHTTPRequestHandler struct {
	*BaseHTTPRequestHandler
	Directory string // 提供服务的目录
}

// NewSimpleHTTPRequestHandler 创建一个新的简单HTTP请求处理器
func NewSimpleHTTPRequestHandler(conn net.Conn, directory string) *SimpleHTTPRequestHandler {
	handler := &SimpleHTTPRequestHandler{
		BaseHTTPRequestHandler: NewBaseHTTPRequestHandler(conn),
		Directory:              directory,
	}
	handler.ProcessMethod = handler // 设置处理方法为自身
	return handler
}

// DoGET 处理GET请求
func (h *SimpleHTTPRequestHandler) DoGET() {
	f, err := h.SendHead()
	if err != nil {
		return
	}
	defer f.Close()

	// 发送文件内容
	io.Copy(h.WFile, f)
	h.WFile.Flush()
}

// DoHEAD 处理HEAD请求
func (h *SimpleHTTPRequestHandler) DoHEAD() {
	f, err := h.SendHead()
	if err != nil {
		return
	}
	f.Close()
}

// DoPOST handles file upload with support for target path
func (h *SimpleHTTPRequestHandler) DoPOST() {
	contentType := h.Headers["Content-Type"]
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil || !strings.HasPrefix(mediaType, "multipart/") {
		h.SendError(BAD_REQUEST, "Content-Type must be multipart/form-data")
		return
	}
	boundary := params["boundary"]
	reader := multipart.NewReader(h.RFile, boundary)
	target := h.TranslatePath(h.Path)
	uploadDir := strings.HasSuffix(h.Path, "/")
	if !uploadDir {
		if info, err := os.Stat(target); err == nil && info.IsDir() {
			uploadDir = true
		}
	}
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			h.SendError(INTERNAL_SERVER_ERROR, "Error reading multipart data")
			return
		}
		var dest string
		if uploadDir {
			filename := filepath.Base(part.FileName())
			if filename == "" {
				continue
			}
			dest = filepath.Join(target, filename)
		} else {
			dest = target
		}
		dst, err := os.Create(dest)
		if err != nil {
			h.SendError(INTERNAL_SERVER_ERROR, "Cannot create file")
			return
		}
		if _, err := io.Copy(dst, part); err != nil {
			dst.Close()
			h.SendError(INTERNAL_SERVER_ERROR, "Error saving file")
			return
		}
		dst.Close()
		if !uploadDir {
			break
		}
	}
	h.SendResponse(OK, "Upload successful")
	h.EndHeaders()
}

// SendHead 发送文件头信息
func (h *SimpleHTTPRequestHandler) SendHead() (*os.File, error) {
	path := h.TranslatePath(h.Path)
	var f *os.File
	var err error
	var needClose bool = true // 标记是否需要关闭文件

	// 确保在错误时关闭已打开的文件
	defer func() {
		if needClose && f != nil {
			f.Close()
		}
	}()

	// 第一阶段：路径检查
	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			h.SendError(NOT_FOUND, "File not found")
		} else {
			h.SendError(INTERNAL_SERVER_ERROR, "File error")
		}
		return nil, err
	}

	// 第二阶段：目录处理
	if stat.IsDir() {
		// 检查是否需要添加尾部斜杠
		if !strings.HasSuffix(h.Path, "/") {
			// 重构完整URL（保留查询参数）
			rawURL := h.RequestLine
			if idx := strings.Index(rawURL, " "); idx != -1 {
				rawURL = rawURL[:idx]
			}
			parsedURL, _ := url.Parse(rawURL)
			parsedURL.Path += "/"
			newURL := parsedURL.String()

			h.SendResponse(MOVED_PERMANENTLY, "")
			h.SendHeader("Location", newURL)
			h.SendHeader("Content-Length", "0")
			h.EndHeaders()
			return nil, nil
		}

		// 查找索引文件
		foundIndex := false
		for _, index := range []string{"index.html", "index.htm"} {
			indexPath := filepath.Join(path, index)
			if fs, err := os.Stat(indexPath); err == nil && !fs.IsDir() {
				path = indexPath
				foundIndex = true
				break
			}
		}

		if !foundIndex {
			// 列目录处理
			return h.ListDirectory(path)
		}

		// 重新获取文件状态（因为path可能指向index文件）
		if stat, err = os.Stat(path); err != nil {
			h.SendError(NOT_FOUND, "File not found")
			return nil, err
		}
	}

	// 第三阶段：路径验证
	if strings.HasSuffix(path, "/") || strings.HasSuffix(path, string(filepath.Separator)) {
		h.SendError(NOT_FOUND, "File not found")
		return nil, os.ErrNotExist
	}

	// 第四阶段：缓存验证
	if ims := h.Headers["If-Modified-Since"]; ims != "" {
		modTime := stat.ModTime().UTC().Truncate(time.Second)
		if t, err := time.Parse(time.RFC1123, ims); err == nil {
			t = t.UTC()
			if !modTime.After(t) {
				h.SendResponse(NOT_MODIFIED, "")
				h.EndHeaders()
				return nil, nil
			}
		}
	}

	// 第五阶段：打开文件
	f, err = os.Open(path)
	if err != nil {
		h.SendError(NOT_FOUND, "File not found")
		return nil, err
	}

	// 第六阶段：发送头信息
	h.SendResponse(OK, "")
	h.SendHeader("Content-Type", h.GuessType(path))
	h.SendHeader("Content-Length", strconv.FormatInt(stat.Size(), 10))
	h.SendHeader("Last-Modified", stat.ModTime().UTC().Format(time.RFC1123))
	h.EndHeaders()

	needClose = false // 调用者需要负责关闭文件
	return f, nil
}

// CGIHTTPRequestHandler 实现CGI HTTP请求处理器
type CGIHTTPRequestHandler struct {
	*SimpleHTTPRequestHandler
	CGIDirectories []string // CGI脚本目录列表
}

// NewCGIHTTPRequestHandler 创建一个新的CGI HTTP请求处理器
func NewCGIHTTPRequestHandler(conn net.Conn, server *HTTPServer, directory string) *CGIHTTPRequestHandler {
	simpleHandler := NewSimpleHTTPRequestHandler(conn, directory)
	return &CGIHTTPRequestHandler{
		SimpleHTTPRequestHandler: simpleHandler,
		CGIDirectories:           []string{"/cgi-bin", "/htbin"},
	}
}

// IsCGIScript 检查路径是否为CGI脚本
func (h *CGIHTTPRequestHandler) IsCGIScript(path string) bool {
	dir, _ := filepath.Split(path)
	dir = filepath.ToSlash(dir)
	for _, cgiDir := range h.CGIDirectories {
		if strings.HasPrefix(dir, cgiDir) {
			return true
		}
	}
	return false
}

// DoGET 处理GET请求
func (h *CGIHTTPRequestHandler) DoGET() {
	if h.IsCGIScript(h.Path) {
		h.RunCGI()
	} else {
		h.SimpleHTTPRequestHandler.DoGET()
	}
}

// DoPOST 处理POST请求
func (h *CGIHTTPRequestHandler) DoPOST() {
	if h.IsCGIScript(h.Path) {
		h.RunCGI()
	} else {
		h.SimpleHTTPRequestHandler.DoPOST()
	}
}

// RunCGI 运行CGI脚本
func (h *CGIHTTPRequestHandler) RunCGI() {
	// 注意：这是一个简化的实现，实际上需要更多的安全检查和错误处理
	h.SendError(NOT_IMPLEMENTED, "CGI script execution not implemented")
	// 在实际实现中，这里应该执行CGI脚本并处理其输出
}

// TranslatePath 将URL路径转换为文件系统路径，确保路径安全
func (h *SimpleHTTPRequestHandler) TranslatePath(urlPath string) string {
	// 移除查询参数和锚点
	if idx := strings.Index(urlPath, "?"); idx != -1 {
		urlPath = urlPath[:idx]
	}
	if idx := strings.Index(urlPath, "#"); idx != -1 {
		urlPath = urlPath[:idx]
	}

	// 判断原始路径是否以斜杠结尾（去除右侧空白后）
	trimmedRawPath := strings.TrimRight(urlPath, " \t\n\r")
	trailingSlash := strings.HasSuffix(trimmedRawPath, "/")

	// 解码URL路径
	decodedPath, err := url.PathUnescape(urlPath)
	if err != nil {
		h.SendError(BAD_REQUEST, "Bad URL encoding")
		return ""
	}

	// 规范化路径并确保绝对路径
	cleanPath := path.Clean(decodedPath)
	if !strings.HasPrefix(cleanPath, "/") {
		cleanPath = "/" + cleanPath
	}

	// 分割路径并过滤无效组件
	parts := strings.Split(cleanPath, "/")
	validParts := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue // 跳过空组件（如双斜杠）
		}

		// 跳过包含文件系统分隔符或特殊目录的组件
		if strings.Contains(part, string(filepath.Separator)) ||
			part == "." || part == ".." {
			continue
		}
		validParts = append(validParts, part)
	}

	// 构建文件系统路径
	fsPath := h.Directory
	for _, part := range validParts {
		fsPath = filepath.Join(fsPath, part)
	}

	// 保留原始路径的尾部斜杠语义
	if trailingSlash {
		fsPath += string(filepath.Separator)
	}

	return fsPath
}

// GuessType 猜测文件的MIME类型
func (h *SimpleHTTPRequestHandler) GuessType(path string) string {
	contentType := mime.TypeByExtension(filepath.Ext(path))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return contentType
}

// ListDirectory 列出目录内容
func (h *SimpleHTTPRequestHandler) ListDirectory(_path string) (*os.File, error) {
	// 读取目录内容
	dir, err := os.Open(_path)
	if err != nil {
		h.SendError(NOT_FOUND, "No permission to list directory")
		return nil, err
	}

	// 获取目录项
	entries, err := dir.Readdir(-1)
	dir.Close()
	if err != nil {
		h.SendError(INTERNAL_SERVER_ERROR, "Error reading directory")
		return nil, err
	}

	// 排序目录项
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	// 生成HTML页面
	displayPath := html.EscapeString(h.Path)
	var buf strings.Builder

	buf.WriteString(fmt.Sprintf("<!DOCTYPE HTML>\n"))
	buf.WriteString(fmt.Sprintf("<html>\n<head>\n"))
	buf.WriteString(fmt.Sprintf("<meta charset=\"utf-8\">\n"))
	buf.WriteString(fmt.Sprintf("<title>Directory listing for %s</title>\n", displayPath))
	buf.WriteString(fmt.Sprintf("</head>\n<body>\n"))
	buf.WriteString(fmt.Sprintf("<h1>Directory listing for %s</h1>\n", displayPath))
	buf.WriteString(fmt.Sprintf("<hr>\n<ul>\n"))

	// 添加返回上级目录的链接
	if h.Path != "/" {
		buf.WriteString(fmt.Sprintf("<li><a href=\"%s\">../</a></li>\n", path.Dir(h.Path)+"/"))
	}

	// 添加目录项
	for _, entry := range entries {
		name := entry.Name()
		link := url.PathEscape(name)
		if entry.IsDir() {
			link += "/"
			name += "/"
		}
		size := "-"
		if !entry.IsDir() {
			size = strconv.FormatInt(entry.Size(), 10)
		}
		mtime := entry.ModTime().Format(time.RFC1123)
		buf.WriteString(fmt.Sprintf("<li><a href=\"%s\">%s</a> %s %s</li>\n", link, html.EscapeString(name), size, mtime))
	}

	buf.WriteString(fmt.Sprintf("</ul>\n<hr>\n</body>\n</html>\n"))

	// 发送响应
	h.SendResponse(OK, "")
	h.SendHeader("Content-Type", "text/html; charset=utf-8")
	h.SendHeader("Content-Length", strconv.Itoa(buf.Len()))
	h.EndHeaders()

	// 创建临时文件并写入内容
	tmpFile, err := os.CreateTemp("", "dirlist")
	if err != nil {
		h.SendError(INTERNAL_SERVER_ERROR, "Error creating temporary file")
		return nil, err
	}

	tmpFile.WriteString(buf.String())
	tmpFile.Seek(0, 0)
	return tmpFile, nil
}
