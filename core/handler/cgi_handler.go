package handler

import (
	"fmt"
	"net"
	_ "net/url"
	"os"
	"os/exec"
	_ "path"
	"path/filepath"
	_ "slices"
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

	// 执行CGI脚本
	talklog.Info(gid, "Executing CGI script: %s", scriptFile)
	cmd := exec.Command(scriptFile)
	cmd.Env = env

	// 修改: 不直接写入到连接，而是捕获输出
	output, err := cmd.CombinedOutput()
	if err != nil {
		talklog.Error(gid, "CGI execution failed: %v", err)
		h.SendError(INTERNAL_SERVER_ERROR, fmt.Sprintf("CGI script execution failed: %v", err))
		return
	}

	// 处理CGI脚本输出
	// 检查输出是否包含HTTP头
	h.SendResponse(OK, "")
	h.SendHeader("Content-Type", "text/plain; charset=utf-8")
	h.SendHeader("Transfer-Encoding", "chunked")
	h.EndHeaders()
	
	// 将 output 包装为分块格式
	chunkHeader := fmt.Sprintf("%x\r\n", len(output))
	h.WFile.Write([]byte(chunkHeader))
	h.WFile.Write(output)
	h.WFile.Write([]byte("\r\n"))
	
	// 结束块
	h.WFile.Write([]byte("0\r\n\r\n"))
	h.WFile.Flush()
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
