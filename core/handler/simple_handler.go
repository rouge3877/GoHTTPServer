package handler

import (
	"compress/gzip"
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
			Query:       utils.ParseQuery(h.QueryRaw),
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
	var returnedFile *os.File // Track the file actually returned

	defer func() {
		// If f was opened but not the file ultimately returned (e.g., replaced by tmpF or error occurred), close it.
		if f != nil && f != returnedFile {
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

	// 第二阶段：判断是否是一个对于一个文件夹的请求
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
		for _, index := range []string{"index.html", "index.htm", "index"} {
			indexPath := filepath.Join(path, index)
			if fs, err := os.Stat(indexPath); err == nil && !fs.IsDir() {
				path = indexPath
				foundIndex = true
				break
			}
		}

		if !foundIndex {
			// 列目录处理
			// Make sure ListDirectory returns nil error if it sends response itself
			tmpFile, listErr := h.ListDirectory(path) // ListDirectory now returns the temp file or error
			if listErr != nil {
				// Error already sent by ListDirectory
				return nil, listErr
			}
			if tmpFile != nil {
				// ListDirectory generated content and sent headers
				returnedFile = tmpFile // Mark tmpFile as returned
				return tmpFile, nil    // Return the temp file with HTML listing
			}
			// If ListDirectory didn't return a file, it means it found an index file.
			// Need to re-stat the index file path.
			// Find index file again (logic duplicated from original ListDirectory check)
			foundIndex = false
			for _, index := range []string{"index.html", "index.htm", "index"} {
				indexPath := filepath.Join(path, index)
				if fs, err := os.Stat(indexPath); err == nil && !fs.IsDir() {
					path = indexPath // Update path to the index file
					stat = fs        // Update stat to the index file
					foundIndex = true
					break
				}
			}
			if !foundIndex {
				// Should have been handled by ListDirectory returning a file
				h.SendError(INTERNAL_SERVER_ERROR, "Index file logic error")
				return nil, os.ErrNotExist // Or a more specific error
			}
			// Proceed to handle the found index file
		}
	}

	// 第三阶段：路径验证 (Check again after potential index file resolution)
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
				return nil, nil // Return nil, nil for NOT_MODIFIED
			}
		}
	}

	// 第五阶段：打开文件
	f, err = os.Open(path)
	if err != nil {
		h.SendError(NOT_FOUND, "File not found")
		return nil, err
	}

	// 第六阶段: 尝试 Gzip 压缩 (如果适用)
	h.IsGzip = config.Cfg.Server.IsGzip && h.IsGzip

	if h.IsGzip {
		tmpF, err := os.CreateTemp("", "gzip*")
		if err == nil {
			// Setup deferred cleanup for the temp file
			cleanupTemp := true // Flag to control cleanup
			defer func() {
				if cleanupTemp {
					tmpF.Close()
					os.Remove(tmpF.Name())
				}
			}()

			gw := gzip.NewWriter(tmpF)
			// Use TeeReader to write to gzip writer while allowing original file to be read later if needed? No, copy directly.
			_, copyErr := io.Copy(gw, f) // Copy from original file 'f'
			closeErr := gw.Close()       // Close the gzip writer *before* seeking/stating

			if copyErr == nil && closeErr == nil {
				// Seek to start for reading
				if _, seekErr := tmpF.Seek(0, io.SeekStart); seekErr == nil {
					// Get stat of the compressed file
					if tmpStat, statErr := tmpF.Stat(); statErr == nil {
						// Compression successful! Send headers for compressed file.
						h.SendResponse(OK, "")
						h.SendHeader("Content-Encoding", "gzip")
						h.SendHeader("Content-Type", h.GuessType(path))                          // Use original path for type
						h.SendHeader("Content-Length", strconv.FormatInt(tmpStat.Size(), 10))    // Compressed size
						h.SendHeader("Last-Modified", stat.ModTime().UTC().Format(time.RFC1123)) // Original mod time
						h.EndHeaders()

						// We are returning tmpF. Prevent its deferred cleanup.
						cleanupTemp = false
						// The original file 'f' is no longer needed by this function or its caller. Close it now.
						f.Close()
						f = nil             // Ensure the main defer doesn't try to close it again
						returnedFile = tmpF // Mark tmpF as the returned file
						return tmpF, nil    // Return the compressed temp file
					} else {
						talklog.Error(talklog.GID(), "Error stating compressed temp file: %v", statErr)
					}
				} else {
					talklog.Error(talklog.GID(), "Error seeking compressed temp file: %v", seekErr)
				}
			} else {
				talklog.Error(talklog.GID(), "Error during gzip compression: copyErr=%v, closeErr=%v", copyErr, closeErr)
				// Need to rewind original file 'f' as copy might have consumed it
				if _, seekErr := f.Seek(0, io.SeekStart); seekErr != nil {
					talklog.Error(talklog.GID(), "Error rewinding original file after failed compression: %v", seekErr)
					h.SendError(INTERNAL_SERVER_ERROR, "Failed to process file")
					// The main defer will close f
					return nil, seekErr
				}
			}
			// If we reach here, compression failed. Fall through to send original file.
			// The deferred tmpF cleanup will execute.
		} else {
			talklog.Error(talklog.GID(), "Error creating temp file for gzip: %v", err)
			// Fall through to send original file.
		}
	}

	// 第七阶段：发送未压缩文件的头信息 (if gzip not applicable or failed)
	h.SendResponse(OK, "")
	h.SendHeader("Content-Type", h.GuessType(path))
	h.SendHeader("Content-Length", strconv.FormatInt(stat.Size(), 10)) // Original size
	h.SendHeader("Last-Modified", stat.ModTime().UTC().Format(time.RFC1123))
	h.EndHeaders()

	returnedFile = f // Mark f as the returned file
	return f, nil    // Return the original file
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
// Returns the temporary file containing the HTML listing, or nil if an index file was found/handled.
// Sends error responses internally.
func (h *SimpleHTTPRequestHandler) ListDirectory(path string) (*os.File, error) {
	// Check for index files first (moved from SendHead)
	for _, index := range []string{"index.html", "index.htm", "index"} {
		indexPath := filepath.Join(path, index)
		if fs, err := os.Stat(indexPath); err == nil && !fs.IsDir() {
			// Found an index file, let SendHead handle it.
			// Indicate success but no file to return from here.
			return nil, nil
		}
	}

	// No index file found, proceed with listing
	d, err := os.Open(path)
	if err != nil {
		h.SendError(INTERNAL_SERVER_ERROR, "Cannot open directory for listing")
		return nil, err
	}
	defer d.Close()

	files, err := d.Readdir(-1)
	if err != nil {
		h.SendError(INTERNAL_SERVER_ERROR, "Error reading directory")
		return nil, err
	}

	// Sort file list
	sort.Slice(files, func(i, j int) bool { return files[i].Name() < files[j].Name() })

	// Build HTML page
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

	// Create temporary file and write HTML content
	tmpFile, err := os.CreateTemp("", "dirlist*.html")
	if err != nil {
		h.SendError(INTERNAL_SERVER_ERROR, "Error creating temporary file for listing")
		return nil, err
	}

	if _, err := tmpFile.WriteString(html); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		h.SendError(INTERNAL_SERVER_ERROR, "Error writing directory listing")
		return nil, err
	}
	if _, err := tmpFile.Seek(0, 0); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		h.SendError(INTERNAL_SERVER_ERROR, "Error seeking in directory listing file")
		return nil, err
	}

	// Send response headers *before* returning the file
	h.SendResponse(OK, "")
	h.SendHeader("Content-Type", "text/html; charset=utf-8")
	h.SendHeader("Content-Length", strconv.Itoa(len(html)))
	// Add Last-Modified? Maybe based on directory mod time? For now, omit.
	h.EndHeaders()

	// Return the temporary file; caller (SendHead/DoGET) is responsible for closing it.
	return tmpFile, nil
}
