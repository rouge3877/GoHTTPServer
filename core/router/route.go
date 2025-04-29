package router

// 注册路由
func (r *Router) RegisterRoute(method, pattern, description string, handler HandlerFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.routes = append(r.routes, RouteEntry{
		Method:      method,
		Pattern:     pattern,
		Description: description,
		Handler:     handler,
	})
}

// 匹配路由
func (r *Router) MatchRoute(method, path string) (HandlerFunc, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, route := range r.routes {
		if route.Method == method && route.Pattern == path {
			return route.Handler, true
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
func (g *Group) RegisterRoute(method, pattern, disposition string, handler HandlerFunc) {
	fullPath := g.prefix + pattern
	g.route.RegisterRoute(method, fullPath, disposition, handler)
}

// 热更新
func (r *Router) Update(method, pattern, newDisposition string, newHandler HandlerFunc) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, route := range r.routes {
		if route.Method == method && route.Pattern == pattern {
			r.routes[i].Handler = newHandler
			r.routes[i].Description = newDisposition
			return true
		}
	}
	return false
}

// 列出所有路由
func (r *Router) ListRoutes() []RouteEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 拷贝一份返回，避免外部修改
	result := make([]RouteEntry, len(r.routes))
	copy(result, r.routes)
	return result
}
