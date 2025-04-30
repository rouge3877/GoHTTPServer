#!/usr/bin/env python3
# logout.py
import os
from http.cookies import SimpleCookie

SESSION_DB = '/tmp/sessions.txt'

# 清除Session
cookie = SimpleCookie()
if 'HTTP_COOKIE' in os.environ:
    cookie.load(os.environ['HTTP_COOKIE'])
    session_id = cookie.get('session_id').value if 'session_id' in cookie else None
    
    if session_id and os.path.exists(SESSION_DB):
        with open(SESSION_DB, 'r') as f:
            sessions = [line for line in f if not line.startswith(session_id)]
        
        with open(SESSION_DB, 'w') as f:
            f.writelines(sessions)

# 清除Cookie
cookie = SimpleCookie()
cookie['session_id'] = ''
cookie['session_id']['expires'] = 'Thu, 01 Jan 1970 00:00:00 GMT'
cookie['session_id']['path'] = '/'
print(cookie.output())
print("Content-Type: text/html; charset=utf-8\r\n\r\n")

print("""
<html>
<body>
    <h1>已退出登录</h1>
    <a href="./login.html">重新登录</a>
</body>
</html>
""")