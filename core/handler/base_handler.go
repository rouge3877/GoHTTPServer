package handler

import (
	"bufio"
	"fmt"
	"html"
	"io"
	"net"
	"net/textproto"
	"os"
	"strconv"
	"strings"
	"time"

	globalconfig "github.com/Singert/xjtu_cnlab/core/global_config"
	"github.com/Singert/xjtu_cnlab/core/talklog"
)

type ProcessMethod interface {
	DoGET()
	DoHEAD()
	DoPOST()
	DoPUT()
	DoDELETE()
	DoOPTIONS()
	SendHead() (*os.File, error)
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
	clientAddr := ""
	if addr, ok := conn.RemoteAddr().(*net.TCPAddr); ok {
		clientAddr = addr.IP.String()
	}
	return &BaseHTTPRequestHandler{
		Conn:                  conn,
		RFile:                 bufio.NewReader(conn),
		WFile:                 bufio.NewWriter(conn),
		CloseConnection:       true,
		ServerVersion:         globalconfig.GoHTTPServerName() + "/" + strings.Split(globalconfig.GoHTTPServerVersion(), " ")[0],
		SysVersion:            "Go/" + strings.Split(globalconfig.GoVersion(), " ")[0],
		ErrorMessageFormat:    defaultErrorMessageFormat,
		ErrorContentType:      defaultErrorContentType,
		ProtocolVersion:       globalconfig.GlobalConfig.Server.Proto,
		DefaultRequestVersion: "HTTP/1.1",
		HeadersBuffer:         make([][]byte, 0),
		ClientAddress:         clientAddr,
	}
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
		gid := talklog.GID()
		talklog.SetPrefix(gid, "HTTP")
		talklog.Info(gid, "New request from %s", h.ClientAddress)
		if h.RFile == nil {
			talklog.Error(gid, "RFile is nil")
			h.CloseConnection = true
			return
		}

		// 读取请求行
		requestLine, err := h.RFile.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				talklog.Error(gid, "Error reading request line: %v", err)
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
			talklog.Warn(gid, "Parse request failed: %s", requestLine)
			// 错误已经发送，直接返回
			return
		}
		for k, v := range h.Headers {
			talklog.Hdr(gid, k, v)
		}
		talklog.Req(gid, h.Command, h.Path, h.RequestVersion)
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

		talklog.Info(gid, "Request processed successfully: %s %s", h.RequestLine, h.Path)
	}

	// 捕获超时错误
	defer func() {
		if r := recover(); r != nil {
			gid := talklog.GID()
			talklog.Error(gid, "Request timed out: %v", r)
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
	contentLength := 0
	if h.HeadersBuffer != nil {
		contentLength = len(h.HeadersBuffer)
	}
	startTime := time.Now()

	h.LogRequest(code, 0)
	h.SendResponseOnly(code, message)
	h.SendHeader("Server", h.VersionString())
	h.SendHeader("Date", h.DateTimeString())

	duration := time.Since(startTime).Milliseconds()

	talklog.Resp(talklog.GID(), int(code), contentLength, float64(duration))

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