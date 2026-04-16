# GoRemote

远程命令执行系统 - 通过 WebSocket 长连接实现服务端对客户端的远程控制。

## 功能特性

- **远程命令执行** - 服务端向客户端下发命令，客户端执行后返回结果
- **双向文件传输** - 支持服务端与客户端之间的文件上传下载
- **心跳保活** - 客户端定期发送心跳，服务端检测超时断开
- **自动重连** - 客户端支持断线自动重连（指数退避）
- **跨平台** - 支持 Linux 和 Windows

## 项目结构

```
C:\work\GoRemote\
├── cmd/
│   ├── server/main.go       # 服务端入口
│   └── agent/main.go        # 客户端入口
├── internal/
│   ├── common/              # 公共定义（消息结构）
│   ├── server/              # 服务端实现 (Gin + WebSocket)
│   │   ├── server.go       # Gin 引擎与路由
│   │   ├── client.go       # 客户端管理
│   │   ├── handler.go      # 消息处理
│   │   ├── task.go         # 任务管理
│   │   ├── upload.go       # 文件传输管理
│   │   ├── api.go          # HTTP API (Gin handlers)
│   │   ├── config.go       # 配置加载
│   │   └── logger.go       # 日志配置
│   └── agent/              # 客户端实现
│       ├── agent.go        # 主控逻辑
│       ├── connector.go    # WebSocket 连接
│       ├── handler.go      # 消息处理
│       ├── executor.go     # 命令执行
│       ├── reconnect.go    # 重连机制
│       ├── config.go       # 配置加载
│       └── logger.go       # 日志配置
├── config/
│   ├── server.yaml          # 服务端配置
│   └── agent.yaml          # 客户端配置
├── docs/                   # 设计文档
│   ├── docs.go             # Swagger 文档生成代码
│   ├── swagger.json        # Swagger JSON 文档
│   └── swagger.yaml        # Swagger YAML 文档
├── go.mod
└── README.md
```

## 编译

### Windows

```bash
go build -o bin/server.exe ./cmd/server
go build -o bin/agent.exe ./cmd/agent
```

### Linux

```bash
GOOS=linux GOARCH=amd64 go build -o bin/server ./cmd/server
GOOS=linux GOARCH=amd64 go build -o bin/agent ./cmd/agent
```

## 使用

### 1. 修改配置

编辑 `config/server.yaml`：

```yaml
server:
  host: "0.0.0.0"
  port: 8080

auth:
  key: "your-secret-key"  # 修改为你的密钥
```

编辑 `config/agent.yaml`：

```yaml
server:
  address: "localhost:8080"  # 修改为服务端地址

auth:
  key: "your-secret-key"  # 与服务端一致
```

### 2. 启动服务端

```bash
./bin/server.exe --config config/server.yaml
```

### 3. 启动客户端

```bash
./bin/agent.exe --config config/agent.yaml
```

### 4. Swagger UI 接口文档

启动服务后，可通过浏览器访问 Swagger UI 查看和测试所有 API 接口：

```
http://localhost:8080/swagger/index.html
```

### 5. API 接口

| 接口 | 方法 | 说明 |
|------|------|------|
| `/health` | GET | 健康检查 |
| `/api/clients` | GET | 查看在线客户端列表 |
| `/api/clients/:id` | GET | 查看客户端详情 |
| `/api/tasks` | GET | 查看任务列表 |
| `/api/tasks/:id` | GET | 查看任务详情 |
| `/api/exec` | POST | 向客户端下发命令 |
| `/swagger/*` | GET | Swagger UI 接口文档 |

### 6. 下发命令示例

```bash
# 查看客户端列表，获取 client_id
curl http://localhost:8080/api/clients

# 下发命令
curl -X POST http://localhost:8080/api/exec \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "客户端ID",
    "command": "ls -la",
    "timeout": 30
  }'
```

## 消息协议

| type | 方向 | 说明 |
|------|------|------|
| register | C→S | 客户端注册 |
| register_ok | S→C | 注册成功 |
| register_fail | S→C | 注册失败 |
| exec | S→C | 执行命令 |
| result | C→S | 命令结果 |
| ping | C→S | 心跳 |
| pong | S→C | 心跳响应 |
| upload | S→C | 上传文件请求 |
| upload_data | S→C | 上传文件数据 |
| upload_complete | C→S | 上传完成确认 |
| download | C→S | 下载文件请求 |
| download_data | S→C | 下载文件数据 |
| download_complete | S→C | 下载完成确认 |

## 架构

```
                    ┌─────────────┐
                    │  控制中心   │
                    │  (服务端)   │
                    └──────┬──────┘
                           │ WebSocket
           ┌───────────────┼───────────────┐
           │               │               │
    ┌──────┴──────┐ ┌──────┴──────┐ ┌──────┴──────┐
    │  机器A      │ │  机器B      │ │  机器C      │
    │  (客户端)   │ │  (客户端)   │ │  (客户端)   │
    └─────────────┘ └─────────────┘ └─────────────┘
```

## 设计文档

- [服务端设计文档](./docs/server-design.md)
- [客户端设计文档](./docs/agent-design.md)
