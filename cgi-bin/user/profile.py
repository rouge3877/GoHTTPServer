#!/usr/bin/env python3
import os
from http.cookies import SimpleCookie

print("Set-Cookie: session_id=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/")
print("Content-Type: text/html; charset=utf-8\r\n\r\n")

# 配置参数
SESSION_DB = '/tmp/sessions.txt'

# 解析Cookie
username = None
cookie = SimpleCookie()
if 'HTTP_COOKIE' in os.environ:
    cookie.load(os.environ['HTTP_COOKIE'])
    session_id = cookie.get('session_id').value if 'session_id' in cookie else None
    
    # 验证Session
    if session_id and os.path.exists(SESSION_DB):
        with open(SESSION_DB, 'r') as f:
            for line in f:
                parts = line.strip().split(':')
                if parts[0] == session_id:
                    username = parts[1]
                    break

if username:
    print(f"""<html>
<body>
    <h1>欢迎回来，{username}!</h1>
    <p>这是您的个人资料页面。</p>
    <p>您的Session ID是：{session_id}</p>
    <p>您的用户名是：{username}</p>
    <p>您的密码是：<strong>保密</strong></p>
    <a href="./logout.py">退出登录</a>
</body>
</html>""")
else:
    print("""<html>
<body>
    <h1>请先登录</h1>
    <a href="./login.html">前往登录</a>
</body>
</html>""")