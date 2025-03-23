# xjtu_cnlab
# 实验要求：
从零实现一个 HTTP/1.1 Web 服务器，不依赖 Go 的 net/http 高级封装，只使用 net 和 bufio、strings 等基础库来手动解析协议并处理请求。
## 1、基于标准的HTTP/1.1协议（RFC 2616等）
协议支持方面的要求：
### A. 基本要求：
- 支持GET、HEAD和POST三种请求方法
- 支持URI的"%HEXHEX"编码，如对 http://abc.com:80/~smith/ 和 http://ABC.com/%7Esmith/ 两种等价的URI能够正确处理；
- 正确给出应答码（如200，304，100，404，500等）；
- 支持Connection: Keep-Alive和Connection: Close两种连接模式。
### B. 高级要求：
- 支持HTTPS，
- 支持分块传输编码(Chunked Transfer Encoding)，
- 支持gzip等内容编码；
- 支持Cookie（见RFC 2109）的基本机制，实现典型的网站登录；
- 支持基本的缓存处理；
- 支持基于POST方法的文件上传。
## 2、服务器的基本要求包括：
A. 可配置Web服务器的监听地址、监听端口和虚拟路径。
B. 能够多线程处理并发的请求，或采取其他方法正确处理多个并发连接。
C. 对于无法成功定位文件的请求，根据错误原因，作相应错误提示。支持一定的异常情况处理能力。 
D. 服务可以启动和关闭。
E. 在服务器端的日志中记录每一个请求（如IP地址、端口号和HTTP请求命令行，及应答码等）。
## 3、服务器的高级要求是
支持CGI；CGI（Common Gateway Interface）并不是HTTP协议的一部分，而是一个独立的标准，用于定义Web服务器如何与外部程序（如脚本或可执行文件）交互以生成动态内容。CGI的主要目的是允许Web服务器调用外部程序来处理HTTP请求并生成响应。参见RFC 3875。
## 4、测试：
在云服务器上搭建你的WWW服务器，在本人的电脑运行浏览器，测试你的WWW服务器的各项功能。测试中，要求在HTTP应答头中Server域设置为作者的英文名字。



# 项目结构及说明
```
xjtu_cnlab
├── cert # HTTPS证书存放位置（如支持HTTPS）
├── cgi-bin 
│   ├── auth # CGI中的认证相关工具包
│   ├── scripts  # CGI可执行脚本
│   └── utils # CGI公共工具
├── cmd 
│   └── main.go # 入口，启动服务器
├── config
│   ├── config.go # 配置文件读取
│   └── config.yaml # 配置文件
├── docs # 项目文档，可以使用swagger生成
├── global
│   └── global.go 存放全局变量
├── logs # 日志记录
├── README.md
├── server
│   ├── core # 核心网络监听与连接调度
│   │   ├── connection.go # 网络监听、连接管理
│   │   └── server.go # 网络监听、连接管理
│   ├── handlers
│   │   ├── dynamic.go
│   │   ├── handler.go # 统一入口，调用其他handler
│   │   ├── router.go
│   │   ├── static.go
│   │   └── upload.go
│   ├── middleware
│   │   ├── gzip.go
│   │   ├── loging.go
│   │   └── recover.go
│   ├── protocol 协议解析和生成（HTTP协议处理层）
│   │   ├── cookie.go # Cookie处理
│   │   ├── http.go # HTTP协议核心工具函数
│   │   ├── request.go # 请求解析
│   │   └── response.go # 响应构造
│   └── utils # 工具层
│       ├── encoding.go # HTTP头部解析工具
│       └── parser.go # chunked、gzip 等编码解码工具
├── static  # 静态资源根目录（虚拟路径映射）（如果有必要）
│   ├── css
│   ├── images
│   └── js
├──  utils # 全局通用工具库（含url解析）
│   ├── url.go                    # URL解码
│   └── file.go                   # 文件工具（上传保存等）

21 directories, 20 files

```

## 实验要求支持的功能清单

| 功能                                 | 模块/说明                          |
|--------------------------------------|-----------------------------------|
| 支持 GET、HEAD、POST 方法            | `request.go`, `handler.go`       |
| URI 编码（%HEXHEX）支持              | `utils/url.go`                   |
| HTTP 状态码                         | `response.go`                    |
| Keep-Alive / Close                   | `response.go` + 头部处理         |
| 多线程处理                           | `go handleConnection(...)`       |
| HTTPS（可选）                        | 使用 `tls.Listen()` 替代 `net.Listen` |
| 分块传输编码                         | 解析 Transfer-Encoding: chunked  |
| gzip 内容编码                        | 使用 `compress/gzip` 模块 （***？这不对吧***）       |
| Cookie 支持                          | `request.go` / `response.go`     |
| 文件上传                             | `utils/file.go`，处理 multipart/form-data |
| 服务器日志                           | `logger.go`                      |
| 虚拟路径配置                         | `config.go`, `router.go`         |
| CGI 调用                             | `cgi/handler.go`                 |
| 缓存处理                             | 添加缓存相关头，如 `ETag`、`Last-Modified` 等 |


## 分层式设计
```
HTTP服务器
│
├── [核心层 (Core Layer)]
│    │—— TCP监听、连接接受、线程/协程池管理
│
├── [协议层 (Protocol Layer)]
│    │—— HTTP请求解析、响应封装、协议特性处理
│
├── [中间件层 (Middleware Layer)]
│    │—— 日志、异常处理、Gzip压缩、缓存控制等
│
├── [业务处理层 (Handler Layer)]
│    │—— 根据HTTP请求类型调用不同处理器
│
└── [工具层 (Utility Layer)]
    │—— 通用HTTP解析与工具方法封装
```

- 核心层（Core Layer）
    - 监听TCP连接、接收客户端请求；

    - 维护连接池，分配协程处理每个连接；

    - 负责整体服务器的生命周期管理（启动、关闭）；

    - 监控连接健康状态，超时控制；

- 协议层（Protocol Layer）
    - HTTP请求方法解析（GET/POST/HEAD）；

    - URI、Headers、Body解析；

    - HTTP响应构造（状态码、响应头、主体）；

    - 支持Keep-Alive机制；

    - Cookie解析与生成；

- 中间件层（Middleware Layer，可选）
    - 日志记录（访问日志、错误日志）；

    - 全局异常捕获处理；

    - gzip内容压缩、缓存控制（如ETag）；

    - 未来扩展（安全认证、速率控制等）；

- 业务处理层（Handler Layer）
    - 根据URL调用对应的业务处理函数；

    - 静态资源处理；

    - 动态资源调用（CGI脚本执行）；

    - 文件上传处理；

- 工具层（Utility Layer）
    - HTTP头部的辅助解析；

    - 编解码工具（如chunked、gzip压缩与解压缩）；

    - 公共方法（例如MIME类型判断）；

### 模块间调用关系
``` yaml
Client Request
      |
      V
core/server.go  —————>  middleware (Logger, Recovery, Compression...)
      |
      V
protocol/parser.go
      |
      V
routing/router.go
      |
      V
handler/*.go (static, cgi, upload, cookie)
      |
      V
protocol/response.go
      |
      V
core/server.go (send response)
      |
      V
Client Response
```
