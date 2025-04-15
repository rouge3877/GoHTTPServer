package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	_ "time"
)

// HTTPServer 实现基本的HTTP服务器功能
type HTTPServer struct {
	Addr           string             // 服务器地址
	Handler        http.Handler       // 请求处理器
	ServerName     string             // 服务器名称
	ServerPort     int                // 服务器端口
	AllowReuse     bool               // 允许地址重用
	Listener       net.Listener       // 网络监听器
	ShutdownCtx    context.Context    // 关闭上下文
	ShutdownCancel context.CancelFunc // 关闭取消函数
	Wg             sync.WaitGroup     // 等待组，用于等待所有请求处理完成
}

// NewHTTPServer 创建一个新的HTTP服务器
func NewHTTPServer(addr string, handler http.Handler) *HTTPServer {
	ctx, cancel := context.WithCancel(context.Background())
	return &HTTPServer{
		Addr:           addr,
		Handler:        handler,
		AllowReuse:     true,
		ShutdownCtx:    ctx,
		ShutdownCancel: cancel,
	}
}

// ServerBind 绑定服务器地址并存储服务器名称
func (s *HTTPServer) ServerBind() error {
	listener, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}
	s.Listener = listener

	// 获取主机名和端口
	host, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		return err
	}

	// 获取完全限定域名
	hostname, err := net.LookupAddr(host)
	if err != nil || len(hostname) == 0 {
		s.ServerName = host
	} else {
		s.ServerName = hostname[0]
	}

	// 解析端口
	s.ServerPort = 0
	fmt.Sscanf(port, "%d", &s.ServerPort)

	return nil
}

// Serve 开始服务
func (s *HTTPServer) Serve() error {
	if s.Listener == nil {
		if err := s.ServerBind(); err != nil {
			return err
		}
	}

	for {
		conn, err := s.Listener.Accept()
		if err != nil {
			select {
			case <-s.ShutdownCtx.Done():
				return nil
			default:
				fmt.Fprintf(os.Stderr, "Error accepting connection: %v\n", err)
				continue
			}
		}

		s.Wg.Add(1)
		go func(c net.Conn) {
			defer s.Wg.Done()
			defer c.Close()

			// 创建请求处理器并处理请求
			handler := &BaseHTTPRequestHandler{
				Conn:           c,
				Server:         s,
				RequestHandler: s.Handler,
			}
			handler.Handle()
		}(conn)
	}
}

// Shutdown 关闭服务器
func (s *HTTPServer) Shutdown() error {
	s.ShutdownCancel()
	if s.Listener != nil {
		s.Listener.Close()
	}
	s.Wg.Wait()
	return nil
}

// ThreadingHTTPServer 实现支持并发的HTTP服务器
type ThreadingHTTPServer struct {
	*HTTPServer
	DaemonThreads bool // 是否使用守护线程
}

// NewThreadingHTTPServer 创建一个新的支持并发的HTTP服务器
func NewThreadingHTTPServer(addr string, handler http.Handler) *ThreadingHTTPServer {
	return &ThreadingHTTPServer{
		HTTPServer:    NewHTTPServer(addr, handler),
		DaemonThreads: true,
	}
}

// StartServer 启动HTTP服务器的便捷函数
func StartServer(port int, directory string) error {
	addr := fmt.Sprintf(":%d", port)
	handler := http.FileServer(http.Dir(directory))
	server := NewThreadingHTTPServer(addr, handler)

	fmt.Printf("Serving HTTP on 0.0.0.0 port %d (http://localhost:%d/) ...\n", port, port)
	return server.Serve()
}

// DualStackServer 支持双栈(IPv4/IPv6)的HTTP服务器
type DualStackServer struct {
	*ThreadingHTTPServer
	Directory string // 提供服务的目录
}

// NewDualStackServer 创建一个新的支持双栈的HTTP服务器
func NewDualStackServer(addr string, handler http.Handler, directory string) *DualStackServer {
	return &DualStackServer{
		ThreadingHTTPServer: NewThreadingHTTPServer(addr, handler),
		Directory:           directory,
	}
}

// ServerBind 重写绑定方法以支持IPv4/IPv6双栈
func (s *DualStackServer) ServerBind() error {
	config := &net.ListenConfig{}

	// 尝试设置IPV6_V6ONLY=0以支持双栈
	listener, err := config.Listen(context.Background(), "tcp", s.Addr)
	if err != nil {
		return err
	}

	s.Listener = listener

	// 获取主机名和端口
	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		return err
	}

	// 设置服务器名称
	s.ServerName, _ = os.Hostname()

	// 解析端口
	s.ServerPort = 0
	fmt.Sscanf(port, "%d", &s.ServerPort)

	return nil
}
