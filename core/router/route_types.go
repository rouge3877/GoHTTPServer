package router

import (
	"sync"
)

type HandlerFunc func(*Context)

type RouterProvider interface {
	GetRouter() *Router
}

type ConnCount interface {
	WgCounter() int32
}

type Context struct {
	Method      string
	Path        string
	Headers     map[string]string
	Body        []byte
	Query       map[string]string
	Conn        any
	RouterAware RouterProvider
}

// 表示一个路由规则
type RouteEntry struct {
	Method      string
	Pattern     string
	Description string
	Handler     HandlerFunc
}

type RouteEntryJSON struct {
	Method      string `json:"method"`
	Pattern     string `json:"pattern"`
	Description string `json:"description"`
}

type Router struct {
	routes []RouteEntry
	mu     sync.RWMutex
}

type Group struct {
	prefix string
	route  *Router
}

func NewRouter() *Router {
	return &Router{
		routes: make([]RouteEntry, 0),
	}
}
