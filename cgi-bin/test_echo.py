#!/usr/bin/env python3
# test_echo.py - 回显请求参数（GET/POST）
import os
import sys

print("Content-Type: text/plain\n")
if os.environ['REQUEST_METHOD'] == 'GET':
    query = os.environ.get('QUERY_STRING', '')
    print(f"GET 参数: {query}")
elif os.environ['REQUEST_METHOD'] == 'POST':
    content_length = int(os.environ.get('CONTENT_LENGTH', 0))
    post_data = sys.stdin.read(content_length)
    print(f"POST 数据: {post_data}")
