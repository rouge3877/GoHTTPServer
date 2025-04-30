package talklog

import (
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/Singert/xjtu_cnlab/core/utils"
)

var (
	logConfig  LogConfig
	fileHandle *os.File
	logLock    sync.Mutex
)
var (
	prefixLock sync.RWMutex
	logPrefix  map[uint64]string = make(map[uint64]string)
)

// 匹配 ANSI 转义序列
var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// 放在文件开头的全局变量区域
var (
	logBuffer     []string     // 环形缓存日志
	maxBufferSize = 1000       // 最多缓存1000条日志
	bufferLock    sync.RWMutex // 缓存锁
)

func InitLogConfig(lgcfg *LogConfig) {
	logConfig = *lgcfg
	bufferLock.Lock()
	logBuffer = nil
	bufferLock.Unlock()
	if logConfig.LogToFile {
		var err error
		os.MkdirAll(getDir(logConfig.FilePath), 0755)
		fileHandle, err = os.OpenFile(logConfig.FilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Printf("Error opening log file: %v\n", err)
			return
		}
	}
}

func getDir(path string) string {
	lastSlash := strings.LastIndex(path, "/")
	if lastSlash == -1 {
		return "."
	}
	return path[:lastSlash]
}

// GID returns the goroutine ID of the current goroutine.
// This is a workaround for the lack of a built-in way to get the goroutine ID in Go.
// It uses the runtime.Stack function to get the stack trace and extracts the goroutine ID from it.
// Note: This is not a guaranteed way to get the goroutine ID and should be used with caution.
// The goroutine ID is not a stable identifier and may change between different runs of the program.
// It is primarily intended for debugging purposes.
func GID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	var id uint64
	fmt.Sscanf(string(b), "goroutine %d ", &id)
	return id
}

func SetPrefix(gid uint64, prefix string) {
	prefixLock.Lock()
	defer prefixLock.Unlock()
	logPrefix[gid] = prefix
}

func logLine(color, level string, gid uint64, format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	prefix := ""
	if logConfig.WithTime {
		prefix = fmt.Sprintf("[%s] ", time.Now().Format("2006-01-02 15:04:05"))
	}
	// 只为等级上色
	coloredLevel := fmt.Sprintf("%s[%s]%s", color, level, ColorReset)

	prefixLock.RLock()
	p := logPrefix[gid]
	prefixLock.RUnlock()
	modPrefix := ""
	if p != "" {
		modPrefix = fmt.Sprintf("[%s] ", p)
	}

	// 最终格式：时间戳 + 彩色等级 + GID + 正文
	line := fmt.Sprintf("%s%s %s[GID:%d] %s", prefix, coloredLevel, modPrefix, gid, msg)

	logLock.Lock()
	defer logLock.Unlock()

	//console log
	fmt.Println(line)
	// memory buffer
	bufferLock.Lock()
	logBuffer = append(logBuffer, stripANSi(line))
	if len(logBuffer) > maxBufferSize {
		logBuffer = logBuffer[len(logBuffer)-maxBufferSize:]
	}
	bufferLock.Unlock()

	//file
	if logConfig.LogToFile && fileHandle != nil {
		fileHandle.WriteString(stripANSi(line) + "\n")
	}
}

// stripANSi removes ANSI color escape codes from a string.
func stripANSi(s string) string {
	return ansiEscape.ReplaceAllString(s, "")
}
func Boot(gid uint64, format string, a ...any) {
	logLine(ColorCyan, "BOOT", gid, format, a...)
}

func BootDone(duration time.Duration) {
	gid := GID()
	secs := float64(duration.Microseconds()) / 1e6
	logLine(ColorCyan, "BOOT", gid, "服务器启动完成，用时 %.6f 秒", secs)
}

func Info(gid uint64, format string, a ...any) {
	logLine(ColorGreen, "INFO", gid, format, a...)
}

func Warn(gid uint64, format string, a ...any) {
	logLine(ColorYellow, "WARN", gid, format, a...)
}

func Error(gid uint64, format string, a ...any) {
	logLine(ColorRed, "ERROR", gid, format, a...)
}

func Req(gid uint64, method, uri, proto string) {
	logLine(ColorCyan, "REQ", gid, "%s %s %s", method, uri, proto)
}

func Hdr(gid uint64, key, value string) {
	logLine(ColorCyan, "HDR", gid, "%s: %s", key, strings.TrimSpace(value))
}

func Resp(gid uint64, status int) {
	logLine(ColorCyan, "RESP", gid, "%d %s", status, utils.StatusMessages[utils.HTTPStatus(status)])
}

// GetRecentLogs 返回最近的缓存日志（不含颜色）
func GetRecentLogs() []string {
	bufferLock.RLock()
	defer bufferLock.RUnlock()

	// 返回副本以避免外部修改原始内容
	copied := make([]string, len(logBuffer))
	copy(copied, logBuffer)
	return copied
}
