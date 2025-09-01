# gemini-cli-2api

一个用 Go 实现的轻量 HTTP 服务，把 gemini-cli 所使用的 Google Code Assist 的后端能力以“Gemini v1beta 风格”的 HTTP API 暴露出来。支持非流式与 SSE 流式生成、API Key 鉴权、多账户轮询

## 安装

从下载页面找到二进制文件安装: https://github.com/boltrunner/gcli2api/releases

或者自己编译:

```
git clone https://github.com/boltrunner/gcli2api.git
cd gcli2api
go build -o gcli2api .
./gcli2api server --config config.json
```

配置文件参考 [config.json.example](https://github.com/boltrunner/gcli2api/blob/master/config.json.example)

## gemini-cli 多账户登录

把本地的 `~/.gemini` 目录移走, 然后运行 gemini-cli， 此时 gemini-cli 会打开浏览器让你登录。

重复操作, 这样就能得到多个 gemini 的 oauth_creds.json 路径。

## 命令
- `server`：启动 HTTP 服务（启动前会校验配置）
  - 示例：`go run . server -c ./config.json`
- `check`：校验配置文件（包含未知键检测与 authKey 占位符检测）
  - 示例：`go run . check -c ./config.json`

未传子命令时默认等价于 `server`。

## 它能做什么
- 暴露兼容形态的 Gemini 生成接口：
  - `GET /health` 健康检查
  - `GET /v1beta/models` 模型列表（内置 `gemini-2.5-flash` 与 `gemini-2.5-pro`）
  - `POST /v1beta/models/<model>:generateContent` 非流式生成
  - `POST /v1beta/models/<model>:streamGenerateContent` SSE 流式生成
- 支持请求重试（401/429/5xx）与指数退避（带抖动）
- 支持单凭据或多凭据池（流式同一路使用同一凭据）
- 自动发现/缓存 GCP Project ID 到 SQLite（默认 `./data/state.db`）
- 可选 API Key 保护：设置 `authKey` 后，需携带 `Authorization: Bearer <key>` 或 `x-goog-api-key: <key>` 或 `?key=<key>`

## 快速开始
1) 准备 OAuth 凭据 JSON（含可刷新令牌）
2) 放置配置文件 `config.json`（见下）
3) 启动服务：`go run . server -c ./config.json`
4) 健康检查：`curl http://127.0.0.1:8085/health`

## 配置说明（config.json）
字段（小写）：
- `host`（默认 `127.0.0.1`）
- `port`（默认 `8085`）
- `authKey`（可选，若为占位符 `UNSAFE-KEY-REPLACE` 则校验失败）
- `geminiOauthCredsFiles`：凭据文件路径数组（必填）
- `requestMaxRetries`（默认 `3`）
- `requestBaseDelay`（毫秒，默认 `1000`）
- `sqlitePath`（默认 `./data/state.db`）

校验规则：
- 配置包含未知键将报错并指出键名。
- `authKey` 若为示例占位符 `UNSAFE-KEY-REPLACE` 将报错。

示例：
```json
{
  "authKey": "YOUR-SECRET-KEY",
  "port": 6005,
  "host": "127.0.0.1",
  "geminiOauthCredsFiles": [
    "~/.gemini_account1/oauth_creds.json",
    "~/.gemini_account2/oauth_creds.json"
  ],
  "requestMaxRetries": 2,
  "requestBaseDelay": 1000,
  "sqlitePath": "./data/state.db"
}
```

## API 约定与请求格式
- 模型名：`gemini-2.5-flash`、`gemini-2.5-pro`
- 请求体字段遵循 Gemini 风格（`contents`、`generationConfig` 等）

示例（非流式）：
```bash
curl -X POST \
  -H "Authorization: Bearer <你的API_KEY>" \
  -H "Content-Type: application/json" \
  http://127.0.0.1:8085/v1beta/models/gemini-2.5-flash:generateContent \
  -d '{"contents":[{"role":"user","parts":[{"text":"你好，介绍一下你自己"}]}]}'
```

## 构建与运行
- 本地运行：`go run . server -c ./config.json`
- 校验配置：`go run . check -c ./config.json`
- 构建二进制：`go build -o gcli2api .`

## 测试与覆盖率
- 运行测试：`go test ./...`
- 覆盖率报告：`go test ./... -coverprofile=coverage.out && go tool cover -html=coverage.out`

## 认证与安全
- OAuth：通过 `geminiOauthCredsFiles` 提供凭据；运行中令牌自动刷新并可能持久化（0600 权限）。
- API Key：配置 `authKey` 后，受保护接口需携带上述任一方式。

## 致谢

本项目的灵感来自 [justlovemaki/AIClient-2-API](https://github.com/justlovemaki/AIClient-2-API)，并借鉴了其实现和配置方式。

## 已知限制
- 目前内置可用模型为 `gemini-2.5-flash` 与 `gemini-2.5-pro`。
