package handler

import (
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Singert/xjtu_cnlab/core/config"
	"github.com/Singert/xjtu_cnlab/core/router"
	"github.com/Singert/xjtu_cnlab/core/server"
	"github.com/Singert/xjtu_cnlab/core/talklog"
	"github.com/Singert/xjtu_cnlab/core/utils"
)

// SimpleHTTPRequestHandler 实现简单的HTTP请求处理器
type SimpleHTTPRequestHandler struct {
	*BaseHTTPRequestHandler
	Directory string // 提供服务的目录
}

// NewSimpleHTTPRequestHandler 创建一个新的简单HTTP请求处理器
func NewSimpleHTTPRequestHandler(server *server.HTTPServer, conn net.Conn) *SimpleHTTPRequestHandler {
	handler := &SimpleHTTPRequestHandler{
		BaseHTTPRequestHandler: NewBaseHTTPRequestHandler(conn),
		Directory:              config.Cfg.Server.Workdir,
	}
	handler.Server = server
	handler.ProcessMethod = handler // 设置处理方法为自身
	return handler
}

// DoGET 处理GET请求
func (h *SimpleHTTPRequestHandler) DoGET() {
	gid := talklog.GID()
	talklog.Info(gid, "Processing GET request for %s", h.Path)
	talklog.Info(gid, "Finding route for %s", h.Path)
	if handlerFunc, found := h.Server.Router.MatchRoute(h.Command, h.Path); found {
		ctx := &router.Context{
			Method:      "GET",
			Path:        h.Path,
			Headers:     h.Headers,
			Conn:        h.Conn,
			RouterAware: h.Server,
			Query:       utils.ParseQuery(h.RawURL),
		}
		handlerFunc(ctx)
		h.WFile.Flush()
		return
	}
	f, err := h.ProcessMethod.SendHead()
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
	f, err := h.ProcessMethod.SendHead()
	if err != nil {
		return
	}
	f.Close()
}

// DoPOST handles file upload with support for target path
func (h *SimpleHTTPRequestHandler) DoPOST() {
	if handlerFunc, found := h.Server.Router.MatchRoute(h.Command, h.Path); found {
		ctx := &router.Context{
			Method:  "POST",
			Path:    h.Path,
			Headers: h.Headers,
			Conn:    h.Conn,
		}
		handlerFunc(ctx)
		h.WFile.Flush()
		return
	}
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

// TranslatePath 将URL路径转换为文件系统路径
func (h *SimpleHTTPRequestHandler) TranslatePath(path string) string {
	// 去除查询参数
	if idx := strings.Index(path, "?"); idx != -1 {
		path = path[:idx]
	}

	// 解码URL编码
	decodedPath, err := url.PathUnescape(path)
	if err != nil {
		decodedPath = path
	}

	// 规范化路径
	decodedPath = filepath.Clean(decodedPath)

	// 确保路径以 / 开头
	if !strings.HasPrefix(decodedPath, "/") {
		decodedPath = "/" + decodedPath
	}

	// 将URL路径转换为文件系统路径
	result := filepath.Join(h.Directory, decodedPath[1:])
	return result
}

// GuessType 猜测文件的MIME类型
func (h *SimpleHTTPRequestHandler) GuessType(path string) string {
	ext := filepath.Ext(path)
	if ext == "" {
		return "application/octet-stream"
	}

	ext = strings.ToLower(ext)
	if mimeType := mime.TypeByExtension(ext); mimeType != "" {
		return mimeType
	}

	// 默认类型
	return "application/octet-stream"
}

// ListDirectory 列出目录内容
func (h *SimpleHTTPRequestHandler) ListDirectory(path string) (*os.File, error) {
	f, err := os.Open(path)
	if err != nil {
		h.SendError(NOT_FOUND, "File not found")
		return nil, err
	}

	files, err := f.Readdir(-1)
	f.Close()
	if err != nil {
		h.SendError(INTERNAL_SERVER_ERROR, "Error reading directory")
		return nil, err
	}

	// 排序文件列表
	sort.Slice(files, func(i, j int) bool { return files[i].Name() < files[j].Name() })

	// 构建HTML页面
	displayPath := h.Path
	if !strings.HasSuffix(displayPath, "/") {
		displayPath += "/"
	}

	title := "Directory listing for " + displayPath

	html := "<!DOCTYPE HTML>\n"
	html += "<html>\n"
	html += "<head>\n"
	html += "<meta charset=\"utf-8\">\n"
	html += "<title>" + title + "</title>\n"
	html += "</head>\n"
	html += "<body>\n"
	html += "<h1>" + title + "</h1>\n"
	html += "<hr>\n"
	html += "<ul>\n"

	// 添加上级目录链接
	if displayPath != "/" {
		html += "<li><a href=\"../\">../</a></li>\n"
	}

	// 添加文件和目录链接
	for _, file := range files {
		name := file.Name()
		if file.IsDir() {
			name += "/"
		}
		html += "<li><a href=\"" + name + "\">" + name + "</a></li>\n"
	}

	html += "</ul>\n"
	html += "<hr>\n"
	html += "</body>\n"
	html += "</html>\n"

	// 发送响应
	h.SendResponse(OK, "")
	h.SendHeader("Content-Type", "text/html; charset=utf-8")
	h.SendHeader("Content-Length", strconv.Itoa(len(html)))
	h.EndHeaders()

	// 创建临时文件并写入HTML内容
	tmpFile, err := os.CreateTemp("", "dirlist")
	if err != nil {
		h.SendError(INTERNAL_SERVER_ERROR, "Error creating temporary file")
		return nil, err
	}

	tmpFile.WriteString(html)
	tmpFile.Seek(0, 0)

	return tmpFile, nil
}
