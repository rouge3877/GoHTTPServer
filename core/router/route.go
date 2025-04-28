package router

// 注册路由
func (r *Router) RegisterRoute(method, pattern string, handler HandlerFunc) {
	r.routes = append(r.routes, RouteEntry{
		Method:  method,
		Pattern: pattern,
		Hanlder: handler,
	})
}

// 匹配路由
func (r *Router) MatchRoute(method, path string) (HandlerFunc, bool) {
	for _, route := range r.routes {
		if route.Method == method && route.Pattern == path {
			return route.Hanlder, true
		}
	}
	return nil, false
}
