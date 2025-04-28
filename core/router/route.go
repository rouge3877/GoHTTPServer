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

// 注册一个新的路由组
func (r *Router) RegisterGroupRoute(prefix string, fn func(g *Group)) {
	g := &Group{
		prefix: prefix,
		route:  r,
	}
	fn(g)
}

// Group内部注册路由
func (g *Group) RegisterRoute(method, pattern string, handler HandlerFunc) {
	fullPath := g.prefix + pattern
	g.route.RegisterRoute(method, fullPath, handler)
}
