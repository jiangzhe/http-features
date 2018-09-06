# http-examples

## Introduction

- 超文本传输协议
- 请求/应答模式
- 统一资源标识符(URI)
- 无状态
- 元数据支持
- 分布式
- 多媒体

## Request

![http-request](static/http-request.jpg)

## Response

![http-response](static/http-response.jpg)

## Method

- GET 查询指定资源
- POST 创建新资源
- HEAD 查询指定资源（仅返回元数据）
- PUT 
- DELETE
- OPTIONS
- TRACE 
- PATCH 资源的部分更新
- CONNECT 变更为TCP/IP

## Status

- 101 更换协议（例如建立websocket时）
- 200 成功
- 201 成功，资源被创建
- 301 永久重定向
- 302 临时重定向
- 304 资源未被修改（与Last-Modified或Etag联合使用）
- 307 临时重定向（复制原请求，HTTP/1.1）
- 400 错误请求
- 401 未认证
- 403 无权限
- 404 资源不存在
- 500 服务器内部错误
- ...

[wikipedia详细说明](https://en.wikipedia.org/wiki/List_of_HTTP_status_codes)

## Header

- Host
- Connection
- Transfer-Encoding
- Content-Type
- Content-Length
- 

## Cookie


## Connection


## Cache

## Proxy

- 反向代理
- HTTP隧道

![http-tunnel](static/http-tunnel.png)

