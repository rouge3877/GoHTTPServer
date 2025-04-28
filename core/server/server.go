package server

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/user"
	"strconv"
	"sync"
	"time"

	globalconfig "github.com/Singert/xjtu_cnlab/core/global_config"
	"github.com/Singert/xjtu_cnlab/core/talklog"
	"github.com/Singert/xjtu_cnlab/core/handler"
)

// HTTPServer 实现基本的HTTP服务器功能
type HTTPServer struct {
	Addr           string             // 服务器地址
	ServerName     string             // 服务器名称
	ServerPort     int                // 服务器端口
	AllowReuse     bool               // 允许地址重用
	Listener       net.Listener       // 网络监听器
	ShutdownCtx    context.Context    // 关闭上下文
	ShutdownCancel context.CancelFunc // 关闭取消函数
	Wg             sync.WaitGroup     // 等待组，用于等待所有请求处理完成
}

// NewHTTPServer 创建一个新的HTTP服务器
func NewHTTPServer(addr string) *HTTPServer {
	ctx, cancel := context.WithCancel(context.Background())
	return &HTTPServer{
		Addr:           addr,
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
			if globalconfig.GlobalConfig.Server.IsCgi {
				handler := handler.NewCGIHTTPRequestHandler(c)
				handler.Handle()
			} else {
				handler := handler.NewSimpleHTTPRequestHandler(c)
				handler.Handle()
			}
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
func NewThreadingHTTPServer(addr string) *ThreadingHTTPServer {
	return &ThreadingHTTPServer{
		HTTPServer:    NewHTTPServer(addr),
		DaemonThreads: true,
	}
}

// StartServer 启动HTTP服务器的便捷函数
func StartServer() error {
	addr := fmt.Sprintf("%s:%d", globalconfig.GlobalConfig.Server.IPv4, globalconfig.GlobalConfig.Server.Port)
	server := NewThreadingHTTPServer(addr)

	fmt.Printf("Serving HTTP on %s port %d (http://localhost:%d/) ...\n", globalconfig.GlobalConfig.Server.IPv4, globalconfig.GlobalConfig.Server.Port, globalconfig.GlobalConfig.Server.Port)
	talklog.BootDone(time.Since(globalconfig.GlobalConfig.StartTime))
	return server.Serve()
}

// StartDualStackServer 启动双栈HTTP服务器的便捷函数
func StartDualStackServer() error {
	addr := fmt.Sprintf("[%s]:%d", globalconfig.GlobalConfig.Server.IPv6, globalconfig.GlobalConfig.Server.Port)
	server := NewDualStackServer(addr, globalconfig.GlobalConfig.Server.Workdir)

	fmt.Printf("Serving HTTP on [%s] port %d (http://localhost:%d/) at work directory :[%s]...\n",
		globalconfig.GlobalConfig.Server.IPv6, globalconfig.GlobalConfig.Server.Port, globalconfig.GlobalConfig.Server.Port, globalconfig.GlobalConfig.Server.Workdir)
	talklog.BootDone(time.Since(globalconfig.GlobalConfig.StartTime))
	return server.Serve()
}

// DualStackServer 支持双栈(IPv4/IPv6)的HTTP服务器
type DualStackServer struct {
	*ThreadingHTTPServer
	Directory string // 提供服务的目录
}

// NewDualStackServer 创建一个新的支持双栈的HTTP服务器
func NewDualStackServer(addr string, directory string) *DualStackServer {
	return &DualStackServer{
		ThreadingHTTPServer: NewThreadingHTTPServer(addr),
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

// / Serve 开始服务(双栈)
func (s *DualStackServer) Serve() error {
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
			if globalconfig.GlobalConfig.Server.IsCgi {
				handler := handler.NewCGIHTTPRequestHandler(c)
				handler.Handle()
			} else {
				handler := handler.NewSimpleHTTPRequestHandler(c)
				handler.Handle()
			}

		}(conn)
	}
}

// Shutdown 关闭服务器(双栈)
func (s *DualStackServer) Shutdown() error {
	s.ShutdownCancel()
	if s.Listener != nil {
		s.Listener.Close()
	}
	s.Wg.Wait()
	return nil
}

// GetNoBodyUID 获取系统中的nobody用户UID
func GetNoBodyUID() (int, error) {
	nobody, err := user.Lookup("nobody")
	if err != nil {
		return -1, err
	}
	uid, err := strconv.Atoi(nobody.Uid)
	if err != nil {
		return -1, err
	}
	return uid, nil
}

func Executable(path string) bool {
	//获取文件信息
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false
	}
	//检查是否是普通文件
	if fileInfo.Mode().IsRegular() {
		return false
	}
	//检查是否可执行
	mode := fileInfo.Mode().Perm()
	return mode&0111 != 0

}
