#!/usr/bin/env python3
import hashlib
import os
import uuid
import sys

# 配置参数
USER_DB = '/tmp/users.txt'

# CGI输入解析
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

# 设置响应头（遵循profile.py格式）
print("Content-Type: text/html; charset=utf-8\r\n")

# 输入验证
if not username or not password:
    print("""<html><body>
        <h1 style="color:red">错误：用户名和密码不能为空</h1>
        <a href="./register.html">返回注册</a>
    </body></html>""")
    exit()

# 用户存在性检查
user_exists = False
if os.path.exists(USER_DB):
    with open(USER_DB, 'r') as f:
        for line in f:
            if line.startswith(username + ':'):
                user_exists = True
                break

if user_exists:
    print("""<html><body>
        <h1 style="color:red">错误：用户名已存在</h1>
        <a href="./register.html">返回注册</a>
    </body></html>""")
    exit()


# 保存用户数据
with open(USER_DB, 'a') as f:
    f.write(f"{username}:{password}\n")

# 注册成功页面
print(f"""<html>
<head><title>注册成功</title></head>
<body>
    <h1 style="color:green">注册成功！</h1>
    <p>用户名：{username}</p>
    <a href="./login.html">立即登录</a>
</body>
</html>""")