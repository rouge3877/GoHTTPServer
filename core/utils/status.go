package utils

// HTTPStatus 定义HTTP状态码常量
type HTTPStatus int

// HTTP状态码常量定义
const (
	OK                              HTTPStatus = 200
	CREATED                         HTTPStatus = 201
	ACCEPTED                        HTTPStatus = 202
	NO_CONTENT                      HTTPStatus = 204
	RESET_CONTENT                   HTTPStatus = 205
	MOVED_PERMANENTLY               HTTPStatus = 301
	FOUND                           HTTPStatus = 302
	SEE_OTHER                       HTTPStatus = 303
	NOT_MODIFIED                    HTTPStatus = 304
	TEMPORARY_REDIRECT              HTTPStatus = 307
	BAD_REQUEST                     HTTPStatus = 400
	UNAUTHORIZED                    HTTPStatus = 401
	FORBIDDEN                       HTTPStatus = 403
	NOT_FOUND                       HTTPStatus = 404
	METHOD_NOT_ALLOWED              HTTPStatus = 405
	REQUEST_TIMEOUT                 HTTPStatus = 408
	CONFLICT                        HTTPStatus = 409
	GONE                            HTTPStatus = 410
	LENGTH_REQUIRED                 HTTPStatus = 411
	INTERNAL_SERVER_ERROR           HTTPStatus = 500
	NOT_IMPLEMENTED                 HTTPStatus = 501
	BAD_GATEWAY                     HTTPStatus = 502
	SERVICE_UNAVAILABLE             HTTPStatus = 503
	HTTP_VERSION_NOT_SUPPORTED      HTTPStatus = 505
	REQUEST_URI_TOO_LONG            HTTPStatus = 414
	REQUEST_HEADER_FIELDS_TOO_LARGE HTTPStatus = 431
	CONTINUE                        HTTPStatus = 100
)

// 状态码对应的短消息和长消息
var StatusMessages = map[HTTPStatus][]string{
	OK:                              {"OK", "Request fulfilled, document follows"},
	CREATED:                         {"Created", "Document created, URL follows"},
	ACCEPTED:                        {"Accepted", "Request accepted, processing continues"},
	NO_CONTENT:                      {"No Content", "Request fulfilled, nothing follows"},
	RESET_CONTENT:                   {"Reset Content", "Clear input form for further input"},
	MOVED_PERMANENTLY:               {"Moved Permanently", "Object moved permanently"},
	FOUND:                           {"Found", "Object moved temporarily"},
	SEE_OTHER:                       {"See Other", "Object moved"},
	NOT_MODIFIED:                    {"Not Modified", "Document has not changed"},
	BAD_REQUEST:                     {"Bad Request", "Bad request syntax or unsupported method"},
	UNAUTHORIZED:                    {"Unauthorized", "No permission"},
	FORBIDDEN:                       {"Forbidden", "Request forbidden"},
	NOT_FOUND:                       {"Not Found", "Nothing matches the given URI"},
	METHOD_NOT_ALLOWED:              {"Method Not Allowed", "Specified method is invalid for this resource"},
	REQUEST_TIMEOUT:                 {"Request Timeout", "Request timed out"},
	CONFLICT:                        {"Conflict", "Request conflict"},
	GONE:                            {"Gone", "URI no longer exists and has been permanently removed"},
	LENGTH_REQUIRED:                 {"Length Required", "Client must specify Content-Length"},
	INTERNAL_SERVER_ERROR:           {"Internal Server Error", "Server got itself in trouble"},
	NOT_IMPLEMENTED:                 {"Not Implemented", "Server does not support this operation"},
	BAD_GATEWAY:                     {"Bad Gateway", "Invalid responses from another server/proxy"},
	SERVICE_UNAVAILABLE:             {"Service Unavailable", "The server cannot process the request due to a high load"},
	HTTP_VERSION_NOT_SUPPORTED:      {"HTTP Version Not Supported", "Cannot fulfill request"},
	REQUEST_URI_TOO_LONG:            {"Request-URI Too Long", "The URI provided was too long for the server to process"},
	REQUEST_HEADER_FIELDS_TOO_LARGE: {"Request Header Fields Too Large", "The server refused this request because the request header fields are too large"},
	CONTINUE:                        {"Continue", "Client should continue with request"},
}

// 默认错误消息模板
const DefaultErrorMessageFormat = `<!DOCTYPE HTML>
<html lang="en">
    <head>
        <meta charset="utf-8">
        <title>Error response :( </title>
    </head>
    <body>
        <h1>Error response</h1>
        <p>Error code: %d</p>
        <p>Message: %s.</p>
        <p>Error code explanation: %d - %s.</p>
    </body>
</html>
`

const DefaultErrorContentType = "text/html;charset=utf-8"
