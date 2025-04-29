package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"os/user"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/Singert/xjtu_cnlab/core/config"
	"github.com/Singert/xjtu_cnlab/core/router"
	"github.com/Singert/xjtu_cnlab/core/talklog"
)

type ServerInterface interface {
	GetRouter() *router.Router
	Shutdown() error
	ServerBind() error
	GetHTTPServer() *HTTPServer
}

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
	Router         *router.Router     // 路由器，用于处理请求
	ConnCount      atomic.Int32
}

func (s *HTTPServer) WgCounter() int32 {
	return s.ConnCount.Load()
}

func (s *DualStackServer) GetRouter() *router.Router {
	return s.Router
}

func (s *ThreadingHTTPServer) GetRouter() *router.Router {
	return s.Router
}

func (s *HTTPServer) GetRouter() *router.Router {
	return s.Router
}

func (s *HTTPServer) GetHTTPServer() *HTTPServer {
	return s
}

func (s *DualStackServer) GetHTTPServer() *HTTPServer {
	return s.HTTPServer
}

// NewHTTPServer 创建一个新的HTTP服务器
func NewHTTPServer(addr string) *HTTPServer {
	ctx, cancel := context.WithCancel(context.Background())
	return &HTTPServer{
		Addr:           addr,
		AllowReuse:     true,
		ShutdownCtx:    ctx,
		ShutdownCancel: cancel,
		Router:         router.NewRouter(),
	}
}

// ServerBind 绑定服务器地址并存储服务器名称
func (s *ThreadingHTTPServer) ServerBind() error {
	// listener, err := net.Listen("tcp", s.Addr)
	// if err != nil {
	// 	return err
	// }
	// s.Listener = listener

	var (
		listener net.Listener
		err      error
	)
	network := "tcp"
	if config.Cfg.Server.ForceIPV4 {
		network = "tcp4"
		talklog.Boot(talklog.GID(), "强制IPV4")
	}
	if config.Cfg.Server.EnableTLS {
		cert, err := tls.LoadX509KeyPair(config.Cfg.Server.CertFile, config.Cfg.Server.KeyFile)
		if err != nil {
			talklog.Boot(talklog.GID(), "Error loading TLS certificate and key: %v", err)
			return fmt.Errorf("error loading TLS certificate and key: %v", err)
		}

		tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}
		listener, err = tls.Listen(network, s.Addr, tlsConfig)
		if err != nil {
			talklog.Boot(talklog.GID(), "Error starting TLS listener: %v", err)
			return fmt.Errorf("error starting TLS listener: %v", err)
		}
	} else {
		listener, err = net.Listen(network, s.Addr)
		if err != nil {
			talklog.Boot(talklog.GID(), "Error starting listener: %v", err)
			return fmt.Errorf("error starting tcp listener: %v", err)
		}
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
func StartServer() (*ThreadingHTTPServer, error) {
	addr := fmt.Sprintf("%s:%d", config.Cfg.Server.IPv4, config.Cfg.Server.Port)
	server := NewThreadingHTTPServer(addr)
	serverType := ""
	if config.Cfg.Server.EnableTLS {
		serverType = "HTTPS"
	} else {
		serverType = "HTTP"
	}
	fmt.Printf("Serving %s on %s port %d (%s://localhost:%d/) ...\n", serverType, config.Cfg.Server.IPv4, config.Cfg.Server.Port, strings.ToLower(serverType), config.Cfg.Server.Port)
	talklog.BootDone(time.Since(config.Cfg.StartTime))

	return server, nil
}

// StartDualStackServer 启动双栈HTTP服务器的便捷函数
func StartDualStackServer() (*DualStackServer, error) {
	addr := fmt.Sprintf("[%s]:%d", config.Cfg.Server.IPv6, config.Cfg.Server.Port)
	server := NewDualStackServer(addr, config.Cfg.Server.Workdir)
	serverType := ""
	if config.Cfg.Server.EnableTLS {
		serverType = "HTTPS"
	} else {
		serverType = "HTTP"
	}
	fmt.Printf("Serving %s on [%s] port %d (%s://localhost:%d/) at work directory :[%s]...\n",
		serverType, config.Cfg.Server.IPv6, config.Cfg.Server.Port, strings.ToLower(serverType), config.Cfg.Server.Port, config.Cfg.Server.Workdir)
	talklog.BootDone(time.Since(config.Cfg.StartTime))

	return server, nil
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

// 手动构造 socket，控制 socket 选项，再包裹 TLS
func (s *DualStackServer) ServerBind() error {
	var baseListener net.Listener
	var err error

	if config.Cfg.Server.EnableTLS {
		// 先用 ListenConfig 创建底层 socket，确保关闭 IPV6_V6ONLY
		lcfg := &net.ListenConfig{
			Control: func(network, address string, c syscall.RawConn) error {
				var innerErr error
				if network == "tcp6" {
					innerErr = c.Control(func(fd uintptr) {
						innerErr = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IPV6, syscall.IPV6_V6ONLY, 0)
					})
				}
				return innerErr
			},
		}

		baseListener, err = lcfg.Listen(context.Background(), "tcp", s.Addr)
		if err != nil {
			return fmt.Errorf("failed to create dual-stack listener: %w", err)
		}

		// 加载证书
		cert, err := tls.LoadX509KeyPair(config.Cfg.Server.CertFile, config.Cfg.Server.KeyFile)
		if err != nil {
			return fmt.Errorf("failed to load TLS certificate: %w", err)
		}

		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
		}

		// 包装 TLS
		s.Listener = tls.NewListener(baseListener, tlsConfig)
	} else {
		// 普通监听
		lcfg := &net.ListenConfig{}
		s.Listener, err = lcfg.Listen(context.Background(), "tcp", s.Addr)
		if err != nil {
			return fmt.Errorf("failed to start listener: %w", err)
		}
	}

	// 获取主机名和端口
	_, port, err := net.SplitHostPort(s.Listener.Addr().String())
	if err != nil {
		return err
	}
	s.ServerName, _ = os.Hostname()
	fmt.Sscanf(port, "%d", &s.ServerPort)

	return nil
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
