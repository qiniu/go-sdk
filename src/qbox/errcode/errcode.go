package errcode


import (
	"strconv"
)

// --------------------------------------------------------------------

const (
	// 公共模块错误码

	OK		 = 200
	PartialOK	 = 298 // Partial OK

	InvalidArgs      = 400 // Bad input parameter. Error message should indicate which one and why.
	BadToken         = 401 // Token 授权错误（Access Token 超时，用户修改了密码，或输入的密码错）
	BadOAuthRequest  = 403 // Bad OAuth request (wrong consumer token, bad nonce, expired timestamp, …).
	BadRequestMethod = 405 // Request method not expected (generally should be GET or POST).

	TooManyRequests  = 503 // 请求过频繁
	ProcessPanic	 = 597 // 请求处理发生异常
	VersionTooOld    = 598 // 客户端版本过老，支持的协议已经被废除
	FunctionFail 	 = 599 // 请求未完成

	FileModified     = 608 // 文件内容被修改
	NoSuchEntry      = 612 // 指定的 Entry 不存在或已经 Deleted
	EntryExists      = 614 // 要创建的 Entry 已经存在
	BucketIsFull     = 630 // Bucket 数量已经达到上限，创建失败
	NoSuchBucket     = 631 // 指定的 Bucket 不存在
	InvalidCtx        = 701 // 无效的上下文，断点续传校验失败

	// 客户端错误码

	NetworkError		= 9996 // 网络错误(非TimeoutError)。
	TimeoutError		= 9997 // 请求超时。
	UnexceptedResponse 	= 9998 // 非预期的输出。see api.UnexceptedResponse
	InternalError      	= 9999 // 内部错误。see api.InternalError

)

var (
	EInvalidArgs		= Errno(InvalidArgs)
	EBadToken		= Errno(BadToken)
	EBadOAuthRequest	= Errno(BadOAuthRequest)
	EBadRequestMethod	= Errno(BadRequestMethod)
	ETimeoutError		= Errno(TimeoutError)
	EUnexceptedResponse = Errno(UnexceptedResponse)
	EFunctionFail       = Errno(FunctionFail)

	EFileModified = Errno(FileModified)
	ENoSuchEntry  = Errno(NoSuchEntry)
	EEntryExists  = Errno(EntryExists)
	EBucketIsFull = Errno(BucketIsFull)
	ENoSuchBucket = Errno(NoSuchBucket)
	EInvalidCtx   = Errno(InvalidCtx)
	EInternalError = Errno(InternalError)
)




// --------------------------------------------------------------------

type Errno int
type ErrnoMsg struct {
	Errno int
	Msg string
}

func (e Errno) Error() string {
	if msg, ok := ErrString[int(e)]; ok {
		return msg
	}
	return "errno:" + strconv.Itoa(int(e))
}

func RegisterErrno(em []ErrnoMsg) {
	for _, r := range em {
		ErrString[r.Errno] = r.Msg
	}
}

func RegisterError(code int, errstr string) int {
	ErrString[code] = errstr
	return code
}

var ErrString = map[int]string{

	OK: "OK",
	PartialOK: "Partial OK",

	InvalidArgs: "invalid arguments",
	BadToken: "bad token",
	BadOAuthRequest: "bad oauth request",
	BadRequestMethod: "bad request method",

	TooManyRequests: "too many requests",
	ProcessPanic: "process panic",
	VersionTooOld: "version too old",
	FunctionFail: "function fail",

	NetworkError: "network error",
	TimeoutError: "timeout",
	UnexceptedResponse: "unexcepted response",
	InternalError: "internal error",

	FileModified: "file modified",
	NoSuchEntry: "no such file or directory", 
	EntryExists: "the entry exists",
	BucketIsFull: "bucket have been full",
	NoSuchBucket: "no such bucket",
	InvalidCtx: "Invalid resumableput progress",
}

// --------------------------------------------------------------------
