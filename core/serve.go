package core

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/Singert/xjtu_cnlab/core/config"
	"github.com/Singert/xjtu_cnlab/core/handler"
	"github.com/Singert/xjtu_cnlab/core/server"
	"github.com/Singert/xjtu_cnlab/core/talklog"
)

// Serve 开始服务
func Serve(s server.ServerInterface) error {

	if s.GetHTTPServer().Listener == nil {
		if err := s.ServerBind(); err != nil {
			return err
		}
	}

	for {
		conn, err := s.GetHTTPServer().Listener.Accept()
		if err != nil {
			select {
			case <-s.GetHTTPServer().ShutdownCtx.Done():
				return nil
			default:
				fmt.Fprintf(os.Stderr, "Error accepting connection: %v\n", err)
				continue
			}
		}
		talklog.Info(talklog.GID(), "新连接已建立，客户端地址：%s", conn.RemoteAddr().String())

		s.GetHTTPServer().Wg.Add(1)
		go func(c net.Conn) {
			config.IncConn()
			defer config.DecConn()
			defer s.GetHTTPServer().Wg.Done()
			defer c.Close()

			//设置超时
			c.SetDeadline(time.Now().Add(config.Cfg.Server.DeadLine * time.Second))
			// 创建请求处理器并处理请求
			// handler := handler.NewHTTPRequestHandler(s, c, config.Cfg.Server.IsCgi)
			// handler.Handle()
			if config.Cfg.Server.IsCgi {
				// CGI处理
				handler := handler.NewCGIHTTPRequestHandler(s.GetHTTPServer(), c)
				handler.Handle()
			} else {
				// 普通HTTP处理
				handler := handler.NewSimpleHTTPRequestHandler(s.GetHTTPServer(), c)
				handler.Handle()
			}

		}(conn)
	}
}
