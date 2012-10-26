---
title: Golang SDK 使用指南 | 七牛云存储
---

# Golang SDK 使用指南

此 Golang SDK 适用于 go1 版本，基于 [七牛云存储官方API](/v3/api/) 构建。使用此 SDK 构建您的网络应用程序，能让您以非常便捷地方式将数据安全地存储到七牛云存储上。无论您的网络应用是一个网站程序，还是包括从云端（服务端程序）到终端（手持设备应用）的架构的服务或应用，通过七牛云存储及其 SDK，都能让您应用程序的终端用户高速上传和下载，同时也让您的服务端更加轻盈。

七牛云存储 Golang SDK 源码地址：[https://github.com/qiniu/go-sdk](https://github.com/qiniu/go-sdk)

**文档大纲**

- [使用](#Usage)
    - [应用接入](#connection!)
		- [配置文件](#config)
		- [认证](#auth)
			- [数字签名](#digest)
			- [UPTOKEN认证](#uptoken)
		- [创建链接](#establish_connection)
    - [上传文件](#upload)
        - [获取用于上传文件的临时授权凭证](#generate-upload-token)
        - [服务端上传文件](#upload-server-side)
            - [断点续上传](#resumable-upload)
            - [针对 NotFound 场景处理](#upload-file-for-not-found)
        - [客户端直传文件](#upload-client-side)
    - [查看文件属性信息](#stat)
    - [获取文件下载链接（含文件属性信息）](#get)
    - [只获取文件下载链接](#download)
    - [删除指定文件](#delete)
    - [删除所有文件（单个 bucket）](#drop)
    - [批量操作](#batch)
        - [批量获取文件属性信息（含下载链接）](#batch_get)
        - [批量获取文件下载链接](#batch_download)
        - [批量删除文件](#batch_delete)
    - [创建公开外链](#publish)
    - [取消公开外链](#unpublish)
    - [Bucket（资源表）管理](#buckets)
        - [创建 Bucket](#mkbucket)
        - [列出所有 Bucket](#list-all-buckets)
        - [访问控制](#set-protected)
    - [图像处理](#op-image)
        - [查看图片属性信息](#image_info)
        - [查看图片EXIF信息](#image_exif)
        - [获取指定规格的缩略图预览地址](#image_preview_url)
        - [高级图像处理（缩略、裁剪、旋转、转化）](#image_mogrify_preview_url)
        - [高级图像处理（缩略、裁剪、旋转、转化）并持久化](#image_mogrify_save_as)
        - [高级图像处理（水印）](#image-watermarking)
            - [水印准备工作](#watermarking-pre-work)
                - [设置原图保护](#watermarking-set-protected)
                - [设置水印预览图URL分隔符](#watermarking-set-sep)
                - [设置水印预览图规格别名](#watermarking-set-style)
            - [设置水印模板](#watermarking-set-template)
            - [获取水印模板](#watermarking-get-template)

- [贡献代码](#Contributing)
- [许可证](#License)


## 使用

<a name="connection!"></a>

### 应用接入

<a name="config"></a>

#### 配置文件

要接入七牛云存储，您需要拥有一对有效的 Access Key 和 Secret Key 用来进行签名认证。可以通过如下步骤获得：

1. [开通七牛开发者帐号](https://dev.qiniutek.com/signup)
2. [登录七牛开发者自助平台，查看 Access Key 和 Secret Key](https://dev.qiniutek.com/account/keys) 。

在获取到 Access Key 和 Secret Key 之后，将它们写入到你的配置文件里，go-sdk使用的配置文件格式是json，并且提供了读取配置文件的API，如下：

	package api

	type Config struct {
		Host map[string]string `json:"HOST"`
		Access_key string `json:"QBOX_ACCESS_KEY"`
		Secret_key string `json:"QBOX_SECRET_KEY"`
		BlockBits uint `json:"BLOCK_BITS"`
		RPutChunkSize int64 `json:"RPUT_CHUNK_SIZE"`
		RPutRetryTimes int `json:"RPUT_RETRY_TIMES"`
		DataPath string `json:"DataPath"`
		Client string `json:"CLIENT"`
		ClientId string `json:"CLIENT_ID"`
		ClientSecret string `json:"CLIENT_SECRET"`
		RedirectURI string `json:"REDIRECT_URI"`
		AuthorizationEndPoint string `json:"AUTHORIZATION_ENDPOINT"`
		TokenEndPoint string `json:"TOKEN_ENDPOINT"`
	}

配置文件的格式可参考qbox.conf文件，如下

	{
		"HOST": {
			"fs": "https://fs.qbox.me",
			"io": "http://iovip.qbox.me",
			"rs": "http://rs.qbox.me",
			"up": "http://up.qbox.me",
			"eu": "http://eu.qbox.me",
			"pu": "http://pu.qbox.me:10200",
			"uc": "http://uc.qbox.me"
		},
		"QBOX_ACCESS_KEY": "4_odedBxmrAHiu4Y0Q10HPG0NANCf6VAsAjWL_kO",
		"QBOX_SECRET_KEY": "SrRuUVfDX6drVRvpyN88v8xcm9XnMZzlbDfvVfmE",
		"BLOCK_BITS": 22,  
		"RPUT_CHUNK_SIZE": 262144,
		"RPUT_RETRY_TIMES": 2,
		"EXPIRES_TIME": 3600,
		"CLIENT": "/rs-put/",
		"CLIENT_ID": "abcd0c7edcdf914228ed8aa7c6cee2f2bc6155e2",
		"CLIENT_SECRET": "fc9ef8b171a74e197b17f85ba23799860ddf3b9c",
		"REDIRECT_URI": "<RedirectURL>",
		"AUTHORIZATION_ENDPOINT": "<AuthURL>",
		"TOKEN_ENDPOINT": "https://acc.qbox.me/oauth2/token"
	}


配置文件主要的配置项的含义如下：

	HOST 是七牛云存储对外服务的服务器列表
	QBOX_ACCESS_KEY 用户的access_key
	QBOX_SECRET_KEY 用户的secret_key
	BLOCK_BITS 断点续上传时默认的BLOCK块大小，为4M
	RPUT_CHUNK_SIZE 断点续上传时默认的CHUNK块大小，为2M
	RPUT_RETRY_TIMES 断点续上传时，如果CHUNK传输失败，重传的次数

七牛SDK里提供了LoadConfig()函数（qbox/api/api.go）来读取一个配置文件，如下：

	package api

	func LoadConfig(filename string) (c *Config) {
		// ...
	}

<a name="auth"></a>

#### 认证

七牛目前v3版本的API主要使用两种认证方式，digest和uptoken方式认证，其中uptoken只用于上传文件到up服务器时才用，也就是说，只有才使用up的API时才用得到，通常是让客户服务器生成一个uptoken，提供给终端用户进行上传文件。digest可用于对所有的请求认证。

<a name="digest"></a>

##### 数字签名

使用数字签名认证需要用到由七牛颁发的`Access_key`和`Secret_key`，API如下（qbox/auth/digest/digest.go）

	package digest

	func NewTransport(key, secret string, t http.RoundTripper) *Transport {
		// ..
	}

<a name="uptoken"></a>

##### UPTOKEN认证

uptoken 认证通常是由客户方服务器使用Access_key和`Secret_key`生成一个授权的uptoken，然后颁发给最终用户，这样最终用户就可以上传文件到我们的七牛服务器

	package uptoken

	func NewTransport(uptoken string, t http.RoundTripper) *Transport {
		// ...
	}

<a name="establish_connection"></a>

#### 创建链接

认证层创建好之后，我们就可以创建一个已包含认证的连接，七牛提供了各种服务，每个服务分别对应package api下的一个子package，分别提供特定的功能，后面介绍到这些功能时会接触到。

	qbox/api/rs
	qbox/api/eu
	qbox/api/uc
	qbox/api/pub
	qbox/api/up
	qbox/api/image

这些package都提供了New方法用来创建一个链接，API如下：

	func New(c *Config, t http.RoundTripper) (s *Service, err error) {
		// ...
	}

其中 t 就是上面我们创建的认证层，比如，如果要创建一个rs服务的连接，方法如下：

	var c Config
	// ...

	t := digest.NewTransport(c.Access_key, c.Secret_key, nil)
	s, err := rs.New(&c, t)

这样我们就可以调用rs中提供的函数和方法，比如上传一个文件s.Put()等等


### 上传文件

<a name="client-side-put"></a>

#### 生成uptoken

客户端要上传一个文件，首先需要调用 SDK 提供的 `digest.MakeAuthTokenToString` 函数来获取一个经过授权用于临时匿名上传的 `uptoken`，uptoken 是经过数字签名的一组数据信息，用来判断一个请求是否是合法的。

MakeAuthTokenString 函数（src/auth/uptoken/uptoken.go）原型如下：

	package uptoken

	type AuthPolicy struct {
		Scope            string `json:"scope"`
		CallbackUrl      string `json:"callbackUrl"`
		CallbackContentType stirng `json:"callback_body_type"`
		Customer         string `json:"customer"`
		Deadline         uint32 `json:"deadline"` // 截止时间（以秒为单位）
	}


	func MakeAuthTokenString(key, secret string, auth *AuthPolicy) string {
		// ...
	}


**参数**

:Scope
: 必须，字符串类型（String），设定文件要上传到的目标 `bucket`

:CallbackUrl
: 可选，字符串类型（String），用于设置文件上传成功后，七牛云存储服务端要回调客户方的业务服务器地址。

:CallbackContentType
: 可选，字符串类型（String），用于设置文件上传成功后，七牛云存储服务端向客户方的业务服务器发送回调请求的 `Content-Type`。

:Customer
: 可选，字符串类型（String），客户方终端用户（End User）的ID，该字段可以用来标示一个文件的属主，这在一些特殊场景下（比如给终端用户上传的图片打上名字水印）非常有用。

:Deadline
: 可选，数字类型，用于设置上传 URL 的有效期，单位：秒，缺省为 3600 秒，即 1 小时后该上传链接不再有效（但该上传URL在其生成之后的59分59秒都是可用的）。

**返回值**

返回一个字符串类型（String）的用于上传文件用的临时授权 uptoken。

<a name="upload-server-side"></a>

#### 服务端上传文件

通过 rs.Put 方法可在客户方的业务服务器上直接往七牛云存储上传文件。该函数规格如下：

	package rs

	func (s *Service) Put(entryURI, mimeType string, body io.Reader, bodyLength int64) (ret PutRet, code int, err error) {
		// ...
	}


**参数**

:entryURI
: 必须，字符串类型（String）

:mimeType
: 可选，文件类型，若不指定，缺省使用 application/octet-stream

:body
: 必须，io.Reader接口，文件的数据从这里读取

:bodyLength
: 必须，int64类型，文件的长度


**返回值**

上传成功，返回如下一个 Hash：

    {"hash"=>"FgHk-_iqpnZji6PsNr4ghsK5qEwR"}

上传失败，请检查 err 返回值变量。

<a name="resumable-upload"></a>

##### 断点续上传

SDK 提供的 up.Put 方法缺省使用断点续上传。默认情况下，SDK 会自动启用断点续上传的方式来上传超过 4MB 大小的文件。您也可以修改配置文件的BLOCK_BITS配置项来改变默认BLOCK块的大小：

	package up

	func (s *Service) Put(
		entryURI, mimeType string, customer, meta, params string,
		body io.ReaderAt, bodyLength int64,
		progfile string, // if uoload haven't done, save the progress into this file
		chunkNotify, blockNotify func(blockIdx int, prog *BlockputProgress)) (code int, err error) {
			// ...
	}


**参数详解**

:entryURI
: 必须，字符串类型（String）

:mimeType
: 可选，字符串类型（String），若不指定，默认为 application/octet-stream

:customer
: 可选，字符串类型（String），该文件的拥有者，一般在使用水印功能时才有用

:meta
: 可选，字符串类型（String），文件的备注信息

:params
: 可选，字符串类型（String），回调给客户服务器的回调参数

:body
: 必须，io.ReaderAt接口，用以读取文件的数据

:bodyLength
: 必须，int64类型，文件的长度

:progfile
: 可选，字符串类型（String），若指定，会读取其中的进度信息恢复上传，并在上传失败时保存上传进度

:chunkNotify 
: 可选，用户自定义函数，若指定，当上传完一个chunk时，会调用此函数，并传入进度信息

:blockNotify
: 可选，用户自定义函数，同上，粒度更大，只在上传完一个BLOCK时才会调用


<a name="upload-file-for-not-found"></a>

##### 针对 NotFound 场景处理

您可以上传一个应对 HTTP 404 出错处理的文件，当您 [创建公开外链](#publish) 后，若公开的外链找不到该文件，即可使用您上传的“自定义404文件”代替之。要这么做，您只须使用 rs.Put 函数上传一个key为“errno-404”的文件即可。

除了使用 SDK 提供的方法，同样也可以借助七牛云存储提供的命令行辅助工具 [qboxrsctl](https://github.com/qiniu/devtools/tags) 达到同样的目的：

    qboxrsctl put <Bucket> <Key> <LocalFile>

将其中的 <Key> 换作 errno-404 即可。

注意，每个 <Bucket> 里边有且只有一个 errno-404 文件，上传多个，最后的那一个会覆盖前面所有的。

<a name="upload-client-side"></a>

#### 客户端直传文件

客户端上传流程和服务端上传类似，差别在于：客户端直传文件所需的 uptoken 可以选择在客户方的业务服务器端生成，也可以选择在客户方的客户端程序里边生成。选择前者，可以和客户方的业务揉合得更紧密和安全些，比如防伪造请求。

简单来讲，客户端上传流程也分为两步：

1. 获取 uptoken（[用于上传文件的临时授权凭证](#generate-upload-token)）
3. 将这个uptoken作为请求Header里的Authorization字段

如果您的网络程序是从云端（服务端程序）到终端（手持设备应用）的架构模型，且终端用户有使用您移动端App上传文件（比如照片或视频）的需求，可以把您服务器得到的此 `upload_token` 返回给手持设备端的App，然后您的移动 App 可以使用 [七牛云存储 Objective-SDK （iOS）](http://docs.qiniutek.com/v3/sdk/objc/) 或 [七牛云存储 Android-SDK](http://docs.qiniutek.com/v3/sdk/android/) 的相关上传函数或参照 [七牛云存储API之文件上传](http://docs.qiniutek.com/v3/api/io/#upload) 直传文件。这样，您的终端用户即可把数据（比如图片或视频）直接上传到七牛云存储服务器上无须经由您的服务端中转，而且在上传之前，七牛云存储做了智能加速，终端用户上传数据始终是离他物理距离最近的存储节点。当终端用户上传成功后，七牛云存储服务端会向您指定的 `callback_url` 发送回调数据。如果 `callback_url` 所在的服务处理完毕后输出 `JSON` 格式的数据，七牛云存储服务端会将该回调请求所得的响应信息原封不动地返回给终端应用程序。


<a name="stat"></a>

### 查看文件属性信息

	package rs

	type Entry struct {
		Hash string `json:"hash"`
		Fsize int64 `json:"fsize"`
		PutTime int64 `json:"putTime"`
		MimeType string `json:"mimeType"`
	}

    func (s *Service) Stat(entryURI string) (entry Entry, code int, err error) {
		// ...
	}

这个函数可用来查看某个已上传文件的属性信息。其中entryURI表示资源的定位字符串。返回的结构体Entry包含文件的哈希值Hash，文件大小Fsize，文件上传的时间PutTime，以及文件的类型MimeType等。一个返回示例如下：

    {
        "fsize"    => 3053,
        "hash"     => "Fu9lBSwQKbWNlBLActdx8-toAajv",
        "mimeType" => "application/x-ruby",
        "putTime"  => 13372775859344500
    }

要注意putTime的单位是百纳秒

<a name="get"></a>

### 获取文件下载链接（含文件属性信息）

	package rs

	type GetRet struct {
		URL string `json:"url"`
		Hash string `json:"hash"`
		MimeType string `json:"mimeType"`
		Fsize int64 `json:"fsize"`
		Expiry int64 `json:"expires"`
	}

	func (s *Service) Get(entryURI, base, attName string, expires int) (data GetRet, code int, err error) {
		// ...
	}

**参数简介**

:base
: 指定文件内容的基准版本，可选。如果指定了base，那么当服务器端文件已经发生变化时，返回失败。

:attName
: 指定下载时要保存的名称，可选。如果未指定，直接是文件名。

:expires
: 返回的GetRet结构里的URL下载链接的过期时间，可选。


**返回值**

返回值主要是一个GetRet结构。一个类似的返回值如下：

    {
        "fsize"    => 3053,
        "hash"     => "Fu9lBSwQKbWNlBLActdx8-toAajv",
        "mimeType" => "application/x-ruby",
        "url"      => "http://iovip.qbox.me/file/<an-authorized-token>",
        "expires"  => 3600
    }


<a name="delete"></a>

### 删除指定文件

	package rs
	
	func (s *Service) Delete(entryURI string) (code int, err error) {
		// ...
	}

函数删除指定entryURI的文件。


<a name="drop"></a>

### 删除一个Bucket

	package rs

    func (s *Service) Drop(bucket string) (code int, err error) {
		// ...
	}

Drop()方法提供了删除整个 bucket 及其里边的所有 key，以及这些 key 关联的所有文件都将被删除。


<a name="batch"></a>

### 批量操作

	package rs
	
	type BatchRet struct {
		Data interface{} `json:"data"`
		Code int `json:"code"`
		Error string `json:"error"`
	}

	type Batcher struct {
		op []string
		ret []BatchRet
	}

	func (b *Batcher) Reset()
	func (b *Batcher) Len()

	func (b *Batcher) Stat(entryURI string, method string)	
	func (b *Batcher) Get(entryURI string, method string)
	func (b *Batcher) Delete(entryURI string, method string)
	func (b *Batcher) Do()

七牛云存储支持常用的Get/Stat/Delete等命令的批量操作，可以只用一次请求执行多个命令，如批量删除可以使用Delete()方法，获取文件信息可以使用Get()方法等等。

<a name="publish"></a>

### 创建公开外链

	package rs

	func (s *Service) Publish(domain, bucket string) (code int, err error) {
		// ...
	}

Publish()方法可以将您在七牛云存储中的资源表 `bucket` 发布到某个 `domain` 下，`domain` 需要在 DNS 管理里边 CNAME 到 `iovip.qbox.me` 。

这样，用户就可以通过 `http://<domain>/<key>` 来访问资源表 `bucket` 中的文件。键值为 `foo/bar/file` 的文件对应访问 URL 为 `http://<domain>/foo/bar/file`。 另外，`domain` 可以是一个真实的域名，比如 `www.example.com`，也可以是七牛云存储的二级路径，比如 `iovip.qbox.me/example` 。

**参数**

domain
: 必须，字符串类型（String），资源表发布的目标域名，例如：`cdn.example.com`

bucket
: 必须，字符串类型（String），要公开发布的资源表名称。


<a name="unpublish"></a>

### 取消公开外链

	package rs

	func (s *Service) Unpublish(domain string) (code int, err error) {
		// ...
	}

这个方法可以用来取消指定 `bucket` 的在某个 `domain` 域下的所有公开外链访问。

**参数**

domain
: 必须，字符串类型（String），资源表已发布的目标域名名称，例如：`cdn.example.com`


<a name="buckets"></a>

### Bucket（资源表）管理

<a name="mkbucket"></a>

#### 创建 Bucket

	package rs
	
	func (s *Service) Mkbucket(bucket string) (code int, err error) {
		// ...
	}

其中bucket为需要创建的bucket名字

<a name="list-all-buckets"></a>

#### 列出所有 Bucket

    package rs
	
	func (s *Service) Buckets() (bs []string, code int, err error) {
		// ...
	}

函数列出当前登录客户的所有 buckets（资源表）。


<a name="set-protected"></a>

#### 访问控制

	package pub
	
	func (s *Service) AccessMode(bucketName string, mode int) (code int, err error) {
		// ...
	}

AccessMode()方法可以用来设置指定 bucket 的访问属性，如果mode为1，则表示启用保护模式，反之亦然。该函数不常用，一般在特殊场景下会用到。比如给图片打水印时，首先要设置原图保护，禁用公开的图像处理操作，采用水印的特殊图像处理，而保护原图就可以通过该函数操作实现。

<a name="op-image"></a>

### 图像处理

<a name="image_info"></a>

#### 查看图片属性信息

	package image

	type ImageInfo struct {
		MimeType string `json:"format"`
		Width int `json:"width"`
		Height int `json:"height"`
		ColorModel string `json:"colorModel"`
	}

	func (s *Service) Info(url string) (ret ImageInfo, code int, err error) {
		// ...
	}

使用Info()方法，我们可以通过图片的下载链接来获得该张图片的属性信息。其中MimeType为图片格式，Width和Height分别为图片的宽高，colorModel为原始图片的着色模式。一个类似的相应结果如下：

    {
        "format"     => "jpeg",
        "width"      => 640,
        "height"     => 425,
        "colorModel" => "ycbcr"
    }

<a name="image_exif"></a>

#### 查看图片EXIF信息

	package image
	
	type ImageExif struct {
		Value string `json:"val"`
		Type int `json:"type"`
	}
	
	func (s *Service) Exif(url string) (ret map[string]ImageExif, code int, err error) {
		// ...
	}

同Info()方法一样，Exif()可以通过图片的下载链接得到图片的元信息。


<a name="imageview"></a>

#### 缩略图和水印

	package image
	
	func (s *Service) View(w io.Writer, url string, params map[string]string) (code int, err error) {
		// ...
	}

View()方法支持将一个存储在七牛云存储的图片进行缩略、裁剪、旋转和格式转化处理，并将转化后的结果存储到io.Writer里面去。其中params是一个map结构，支持的参数如下

**参数**

:Mode
: 缩略图的模式，其值可为1或者2

:Width
: 指定目标缩略图的宽度，单位：像素（px）

:Height
: 指定目标缩略图的高度，单位：像素（px）

:Quality
: 指定目标缩略图的图像质量，取值范围 1-100

:Format
: 指定目标缩略图的输出格式，取值范围：jpg, gif, png, tif 等图片格式

:Sharpen
: 指定目标缩略图的锐化指数，值为正整数，此数值越大，锐化度越高，图像细节损失越大

:Watermark
: 是否打水印，可选值为 0 或者 1。为 0 时表示不打水印；值为 1 时表示取相应的水印模板进行打水印处理。水印模板设置会在后面介绍。

更详细的用法请参考这里（http://docs.qiniutek.com/v3/api/foimg/），示例用法如下

	p := map[string]string {
		"Mode": "1",
		"Width": "200",
		"Height": "200",
		"Format": "gif",
	}
	code, err := ims.View(f, url1, p)
	if code/100 != 2 {
		// Error handler
	}


#### 高级图像处理（缩略、裁剪、旋转、转化）

	package image

	type ImageHash struct {
		Hash string `json:"hash"`
	}

	func (s *Service) Mogr(ret interface{}, url string, params map[string]string) (code int, err error) {
		// ...
	}

Mogr是七牛支持的高级图像处理方法（imageMogr），能对图片进行缩放、剪裁、旋转、格式化等等，其中url为七牛云存储的图片下载链接，params是一个map结构，保存了图像处理的各种参数，用调用者指定。除此之外，我们还支持将处理后的缩略图作为一个新文件持久化存储到七牛云存储服务器上，这样就可以供后续直接使用而不用每次都传入参数进行图像处理。

params支持的参数列表如下：

:Thumbnail
: 缩略图的规格，可选。

:Gravity
: 位置，可选，其值为 NorthWest, North, NorthEast, West, Center, East, SouthWest, South, SouthEast

:Crop
: 对图片进行剪裁，可选。值同Thumbnail

:Quality
: 图片质量，可选。数值越大图片质量越高

:Rotate
: 对图片进行旋转，可选。其值为旋转的角度。

:Format
: 转换后的图片格式，可选，其值为jpg、gif、png、tif等等

:Orient
: 自动旋转，可选。可以自动根据图片的元信息对图片进行旋转，比如将一个横着的图片矫正。

:SaveAs
: 持久化操作，可选。可以将处理后的图像持久化到七牛云存储里，这样就不必每次都需要做图像处理。

需要注意的是，如果 params 里指定了SaveAs参数，则ret可以传入一个ImageHash结构，这样可以得到文件的哈希值。如果没有指定SaveAs参数，则ret必须是一个io.Writer，这样就可以将处理后的图像保存到这个Writer里去。关于 imageMogr 参数里边的具体含义和使用方式，可以参考这里（http://docs.qiniutek.com/v3/api/foimg/#fo-imageMogr）。

用法示例：

	p := map[string]string {
		"Thumbnail": "!100x100",
		"Gravity": "center",
		"Crop": "!100x100",
		"Quality": "80",
	}
	code, err = ims.Mogr(f, url1, p)
	if code/100 != 2 {
		// Error handler
	}

您可以选择将存放缩略图的 bucket 公开，这样就可以直接以外链的形式访问到缩略图，而不用走API获取下载URL。然后将你的域名，比如说 pic.example.com CNAME 到 iovip.qbox.me ，就可以直接通过 http://pic.example.com/<target_key> 来访问图片文件。


<a name="image-watermarking"></a>

## 高级图像处理（水印）

<a name="watermarking-pre-work"></a>

### 水印准备工作

为了保护用户原图和方便用户访问打过水印之后的图片，在经水印作用之前，需进行以下一些设置：

1. [设置原图保护](#watermarking-set-protected)
2. [设置水印预览图URL分隔符](#watermarking-set-sep)
3. [设置水印预览图规格别名](#watermarking-set-style)

<a name="watermarking-set-protected"></a>

#### 1. 设置原图保护

用户的图片打上水印后，其原图不可见。通过给原图所在的 Bucket（资源表）设置访问控制，可以达到保护原图的目的，详情请参考 [Bucket（资源表）管理之访问控制](set-protected)。

设置原图保护也可以借助七牛云存储提供的命令行辅助工具 [qboxrsctl](https://github.com/qiniu/devtools/tags) 来实现：

    // 为<Bucket>下面的所有图片设置原图保护
    qboxrsctl protected <Bucket> <Protected>

<a name="watermarking-set-sep"></a>

#### 2. 设置水印预览图URL分隔符

没有设置水印前，用户可以通过如下公开链接的形式访问原图（[创建公开外链后的情况下](/v3/api/io/#rs-Publish)）：

    http://<Domain>/<Key>

设置水印后，其原图属性为私有，不能再通过这种形式访问。但是用户可以在原图的 `<Key>` 后面加上“分隔符”，以及相应的水印风格样式来访问打水印后的图片。例如，假设您为用户设定的访问水印图的分隔符为中划线 “-”，那么用户可以通过这种形式来访问打水印后的图片：

    http://<Domain>/<Key>-/imageView/<Mode>/w/<Width>/h/<Height>/q/<Quality>/format/<Format>/sharpen/<Sharpen>/watermark/<HasWatermark>

其中，`HasWatermark` 参数为 `0` （或者没有）表示不打水印，为 `1` 表示给图片打水印。

通过SDK提供的 pub.(*Service).Separator() 方法可以设置水印预览图URL分隔符：

	package pub
	
	func (s *Service) Separator(bucket string, sep string) (code int, err error) {
		// ...
	}

除此之外，同样可以借助七牛云存储提供的命令行辅助工具 [qboxrsctl](https://github.com/qiniu/devtools/tags) 达到同样的目的：

    // 设置预览图分隔符
    qboxrsctl separator <Bucket> <Sep>

<a name="watermarking-set-style"></a>

#### 3. 设置水印预览图友好风格

通过步骤2中所描述的水印预览图 URL 来访问打水印后的图片毕竟较为繁琐，因此可以通过为该水印预览图规格设置“友好风格”的形式来访问。如：

别名（Name） | 规格（Style） | 说明
----------- | ------------ | -------
small.jpg   | imageView/0/w/120/h/90 | 大小为 120x90，不打水印
middle.jpg  | imageView/0/w/440/h/330/watermark/1 | 大小为 440x330，打水印
large.jpg   | imageView/0/w/1280/h/760/watermark/1 | 大小为 1280x760，打水印


SDK 提供了 pub.(*Service).Style() 方法可以定义预览图规格别名，该函数原型如下：

	package pub
	
	func (s *Service) Style(bucket, name, style string) (code int, err error) {
		// ...
	}

其中bucket为bucket的名称，每个bucket都有各自的友好风格，name是友好风格名称，style是友好风格的实际内容。除此之外，同样也可以借助七牛云存储提供的命令行辅助工具 [qboxrsctl](https://github.com/qiniu/devtools/tags) 达到同样的目的：

    // 为 <Buecket> 下面的所有图片设置名为 <Name> 的 <Style>
    qboxrsctl style <Bucket> <Name> <Style>

无论是通过 SDK 提供的方法还是命令行辅助工具操作，在设置完成后，即可通过通过以下方式来访问设定规格后的图片：

	// 其中 “-” 为分隔符，“small.jpg” 为预览图规格别名
	[GET] http://<Domain>/<Key>-small.jpg

	// 其中 “!” 为分隔符，“middle.jpg” 为预览图规格别名
	[GET] http://<Domain>/<Key>!middle.jpg

	// 其中 “@” 为分隔符，“large.jpg” 为预览图规格别名
    [GET] http://<Domain>/<Key>@large.jpg

以上这些设置水印模板前的准备只需操作一次，即可对后续设置的所有水印模板生效。由于是一次性操作，建议使用 qboxrsctl 命令行辅助工具进行相关设置。

**取消水印预览图友好风格**

您也可以为某个水印预览图规格取消“别名”设置，SDK 提供了相应的方法：

	package pub
	
	func (s *Service) Unstyle(bucket, name string) (code int, err error) {
		// ...
	}

同理，这个操作也可以借助七牛云存储提供的命令行辅助工具 [qboxrsctl](https://github.com/qiniu/devtools/tags) 来完成：

    // 取消预览图规格别名为 <Name> 的 Style
    qboxrsctl unstyle <Bucket> <Name>


<a name="watermarking-set-template"></a>

### 设置水印模板

给图片加水印，我们可以通过 eu.(*Service).SetWatermark() 方法来设置水印模板 ，通过该操作，客户方可以设置通用的水印模板，也可以为客户方的每一个终端用户分别设置一个水印模板。该方法的函数原型如下：

	package eu	
	
	type Watermark struct {
		Font      string `json:"font"`
		Fill      string `json:"fill"`
		Text      string `json:"text"`
		Bucket    string `json:"bucket"`
		Dissolve  string `json:"dissolve"`
		Gravity   string `json:"gravity"`
		FontSize  int    `json:"fontsize"`	// 0 表示默认。单位: 缇，等于 1/20 磅
		Dx        int    `json:"dx"`
		Dy        int    `json:"dy"`
	}

	func (s *Service) SetWatermark(customer string, param *Watermark) (code int, err error) {
		// ...
	}

其中Watermark模板结构的各个参数意义如下：

1. `Customer => <EndUserID>`
: 客户方终端用户标识。如果不设置，则表示设置默认水印模板。作为面向终端用户的服务提供商，您可以为不同的用户设置不同的水印模板，只需在设置水印模板的时候传入`Customer`参数。如果该参数未设置，则表示为终端用户设置一个默认模板。举例：假如您为终端用户提供的是一个手机拍照软件，用户拍照后图片存储于七牛云存储服务器。为了给每个用户所上传的图片打上标有该用户用户名的水印，您可以为该用户设置一个水印模板，其水印文字可以是该终端用户的用户名。但如果您未给该终端用户设置模板，那么水印上的所有设置都是默认的（其文字部分可能是你们自己设置的企业标识）。该 `Customer` 和 认证时生成的uptoken时用到的AuthPolicy结构里的 `Customer` 字段含义一致，结合这点，您很容易想明白这是为什么。

2. `Font => <FontName>`
: 为水印上的文字设置一个默认的字体名，可选。

3. `Fontsize => <FontSize>`
: 字体大小，可选，0表示默认，单位: 缇，等于 1/20 磅。

4. `Fill => <FillColor>`
: 字体颜色，可选。

5. `Text => <WatermarkText>`
: 水印文字，必须，图片用 \0 - \9 占位。

6. `Bucket => <ImageFromBucket>`
: 如果水印中有图片，需要指定图片所在的 `RS Bucket` 名称，可选。

7. `Dissolve => <Dissolve>`
: 透明度，可选，字符串，如50%。

8. `Gravity => <Gravity>`
: 位置，可选，字符串，默认为右下角（SouthEast）。可选的值包括：NorthWest、North、NorthEast、West、Center、East、SouthWest、South和SouthEast。

9. `Dx => <DistanceX>`
: 横向边距，可选，默认值为10，单位px。

10. `Dy => <DistanceY>`
: 纵向边距，可选，默认值为10，单位px。


<a name="watermarking-get-template"></a>

### 获取水印模板

同上，获取指定终端用户或者缺省水印模板的函数原型如下：

	package eu

	func (s *Service) GetWatermark(customer string) (ret Watermark, code int, err error) {
		// ...
	}

其中返回的Watermark结构前面我们已经了解过，请参考 [水印模板设置](#watermarking-set-template)。其中 customer 是用户ID



<a name="Contributing"></a>

## 贡献代码

七牛云存储 Ruby SDK 源码地址：[https://github.com/qiniu/ruby-sdk](https://github.com/qiniu/ruby-sdk)

1. 登录 [github.com](https://github.com)
2. Fork [https://github.com/qiniu/ruby-sdk](https://github.com/qiniu/ruby-sdk)
3. 创建您的特性分支 (`git checkout -b my-new-feature`)
4. 提交您的改动 (`git commit -am 'Added some feature'`)
5. 将您的改动记录提交到远程 `git` 仓库 (`git push origin my-new-feature`)
6. 然后到 github 网站的该 `git` 远程仓库的 `my-new-feature` 分支下发起 Pull Request

<a name="License"></a>

## 许可证

Copyright (c) 2012 qiniutek.com

基于 MIT 协议发布:

* [www.opensource.org/licenses/MIT](http://www.opensource.org/licenses/MIT)
