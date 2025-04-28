package router

type HandlerFunc func(Context)

type Context struct {
	Method  string
	Path    string
	Headers map[string]string
	Body    []byte
	Query   map[string]string //要不要用
	Conn    any
}

// 表示一个路由规则
type RouteEntry struct {
	Method  string
	Pattern string
	Hanlder HandlerFunc
}

type Router struct {
	routes []RouteEntry
}

func NewRouter() *Router {
	return &Router{
		routes: make([]RouteEntry, 0),
	}
}
