#!/usr/bin/env python3
# test_env.py - 打印所有环境变量（验证请求信息传递）
import os

print("Content-Type: text/plain\n")  # 必须的空行分隔头部和内容
print("CGI 环境变量：")
for key in sorted(os.environ):
    print(f"{key} = {os.environ[key]}")