# lhsync

一个简单的文件监控同步工具

## 特性
1. 监控同步目录，修改自动同步至服务器
2. 支持配置多个同步目录


## 开始
### 安装
直接下载执行文件 [release](https://github.com/shayin/lhsync/releases/)

### 运行
1. 修改服务端配置 `server.yaml`
2. 修改客户端配置 `client.yaml`
3. 运行客户端 ./lhsync_cli -c client.yaml
4. 运行服务端 ./lhsync_server -c server.yaml