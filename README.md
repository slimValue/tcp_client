### tcp client
> 一个更方便使用tcp client的包

功能点:
 - 支持断线重连
 - 支持依赖注入: 事件event、日志logger、协议codec
 - call获取结果
 
 
使用方式
 
 ```
// 初始化
conn := client.NewTcpClient(addr,
    client.ReconnectOption(),
    client.SetLogger(logger),
    client.CodecOption(customCodec{}),
    client.EventHookOption(NewEvent(logger)),
)

conn.Start()
```