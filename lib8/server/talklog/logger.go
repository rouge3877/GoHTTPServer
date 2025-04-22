package talklog

import (
	"fmt"
	"runtime"
	_ "strings"
)

const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorCyan   = "\033[36m"
)

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

func Info(gid uint64, format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	fmt.Printf("%s[INFO] [GID:%d] %s%s\n", ColorGreen, gid, msg, ColorReset)
}

func Warn(gid uint64, format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	fmt.Printf("%s[WARN] [GID:%d] %s%s\n", ColorYellow, gid, msg, ColorReset)
}

func Error(gid uint64, format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	fmt.Printf("%s[ERROR] [GID:%d] %s%s\n", ColorRed, gid, msg, ColorReset)
}

func Req(gid uint64, method, uri, proto string) {
	msg := fmt.Sprintf("%s %s %s", method, uri, proto)
	fmt.Printf("%s[REQ] [GID:%d] %s%s\n", ColorCyan, gid, msg, ColorReset)
}

func Hdr(gid uint64, key, value string) {
	fmt.Printf("%s[HDR] [GID:%d] %s: %s%s\n", ColorCyan, gid, key, value, ColorReset)
}

func Resp(gid uint64, status int, contentLength int, durationMs float64) {
	fmt.Printf("%s[RESP] [GID:%d] %d OK (Content-Length: %d) - served in %.2fms%s\n",
		ColorCyan, gid, status, contentLength, durationMs, ColorReset)
}
