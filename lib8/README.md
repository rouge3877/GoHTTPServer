# Go HTTP Server

这是一个基于Go语言实现的HTTP服务器，功能等同于Python的http.server模块。

## 功能特点

- 实现了基本的HTTP服务器功能
- 支持并发处理请求（使用goroutine）
- 提供了简单的HTTP请求处理器
- 支持HTTP/1.0和HTTP/1.1协议
- 实现了错误处理和日志记录

## 项目结构

```
.
├── README.md           # 项目说明文档
├── go.mod              # Go模块定义
├── main.go             # 主程序入口
├── server/             # 服务器相关代码
│   ├── server.go       # HTTP服务器实现
│   └── handler.go      # 请求处理器实现
└── utils/              # 工具函数
    └── utils.go        # 辅助函数
```

## 使用方法

### 启动服务器

```bash
go run main.go [options] [port]
```

### 选项

- `-d, --directory`: 指定要提供服务的目录（默认：当前目录）
- `-p, --protocol`: 指定HTTP协议版本（默认：HTTP/1.0）
- `port`: 指定绑定的端口（默认：8000）

## 示例

```go
package main

import (
    "github.com/yourusername/httpserver/server"
)

func main() {
    // 创建并启动HTTP服务器
    server.StartServer(8000, "./")
}
```
# TODO
- [ ] 实现剩余方法
- [ ] 重构cgihandler
- [ ] 整理项目结构
- [ ] 高级功能...
- [ ] 给老师请假
- [ ] 使用接口回调实现多态
