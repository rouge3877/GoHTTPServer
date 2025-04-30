#!/usr/bin/env python3
import os
import sys
import hashlib


# 配置参数
USER_DB = '/tmp/users.txt'
SESSION_DB = '/tmp/sessions.txt'

data = ""
# read everything from stdin and print to stdout:
if os.environ['REQUEST_METHOD'] == 'GET':
    data = os.environ.get('QUERY_STRING', '')
elif os.environ['REQUEST_METHOD'] == 'POST':
    content_length = int(os.environ.get('CONTENT_LENGTH', 0))
    data = sys.stdin.read(content_length)

# 解析POST数据
# username=q&password=q 
username = ""
password = ""

if data:
    # 解析数据
    data = data.split('&')
    for item in data:
        key, value = item.split('=')
        if key == 'username':
            username = value
        elif key == 'password':
            password = value

# 验证用户
def validate_user(username, password):

    if os.path.exists(USER_DB):
        with open(USER_DB, 'r') as f:
            for line in f:
                parts = line.strip().split(':')
                if len(parts) == 2 and parts[0] == username and parts[1] == password:
                    return True
    return False
# 生成Session ID
import uuid
def generate_session_id():
    return str(uuid.uuid4())
# 保存Session
def save_session(session_id, username):
    with open(SESSION_DB, 'a') as f:
        f.write(f"{session_id}:{username}\n")
# 设置Cookie
def set_cookie(session_id):
    print(f"Set-Cookie: session_id={session_id}; Path=/; HttpOnly")
    print("Content-Type: text/html; charset=utf-8\r\n\r\n")

    print(f"""
<html>
<body>
    <h1>欢迎回来，{username}!</h1>
    <a href="./profile.py">查看个人资料</a><br>
    <a href="./logout.py">退出登录</a>
</body>
</html>
""")
    

# 主程序
if validate_user(username, password):
    session_id = generate_session_id()
    save_session(session_id, username)
    set_cookie(session_id)
else:
    print("""
<html>
<body>
    <h1>登录失败</h1>
    <p>用户名或密码错误。</p>
    <a href="./login.html">重新登录</a>
</body>
</html>
""")