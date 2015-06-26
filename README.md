qiniupkg.com/api.v7 (Qiniu Go SDK v7.x)
===============

# 下载

```
go get qiniupkg.com/api.v7
```

# 使用文档

## KODO Storage (七牛对象存储)

* [qiniupkg.com/api.v7/kodo](http://godoc.org/qiniupkg.com/api.v7/kodo)
* [qiniupkg.com/api.v7/kodocli](http://godoc.org/qiniupkg.com/api.v7/kodocli)

如果您是在业务服务器（服务器端）调用七牛云存储的服务，请使用 [qiniupkg.com/api.v7/kodo](http://godoc.org/qiniupkg.com/api.v7/kodo)。

如果您是在客户端（比如：Android/iOS 设备、Windows/Mac/Linux 桌面环境）调用七牛云存储的服务，请使用 [qiniupkg.com/api.v7/kodocli](http://godoc.org/qiniupkg.com/api.v7/kodocli)。注意，在这种场合下您不应该在任何地方配置 AccessKey/SecretKey。泄露 AccessKey/SecretKey 如同泄露您的用户名/密码一样十分危险，会影响您的数据安全。
