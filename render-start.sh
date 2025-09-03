#!/bin/sh

# 如果任何命令失败，立即退出脚本
set -e

echo "==> INFO: Preparing configuration from environment variables..."

# 检查环境变量是否存在，提供更清晰的错误提示
if [ -z "$G2A_CONFIG_CONTENT" ] || [ -z "$G2A_AUTH_KEY" ]; then
  echo "==> FATAL: G2A_CONFIG_CONTENT or G2A_AUTH_KEY environment variables are not set."
  exit 1
fi

# 从环境变量动态生成配置文件
echo "$G2A_CONFIG_CONTENT" | sed "s/WILL_BE_SET_BY_ENV_VAR/$G2A_AUTH_KEY/" > /app/config.json

echo "==> INFO: Configuration successfully written to /app/config.json."
echo "==> INFO: Starting gcli2api server..."

# 使用 exec 启动主程序，这是容器中的标准做法，能确保信号正确传递
exec /app/gcli2api server -c /app/config.json
