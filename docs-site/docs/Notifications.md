# 🔔 通知渠道配置（Apprise）

📦 支持 100+ 通知渠道，通过 [Apprise](https://github.com/sinspired/apprise_vercel) 发送通知。

- 中文文档镜像：[文档](https://sinspired.github.io/apprise_vercel/)

## 🌐 Vercel 部署

点击下方按钮，一键部署到你的 `Vercel` 账户：

[![Deploy with Vercel](https://vercel.com/button)](https://vercel.com/new/clone?repository-url=https://github.com/sinspired/apprise_vercel)

部署后获取 API 链接，如 `https://projectName.vercel.app/notify`。

建议为 Vercel 项目设置自定义域名（国内访问 Vercel 可能受限）。

## 🐳 Docker 部署（不支持 arm/v7）

```bash
# 基础运行
docker run --name apprise -p 8000:8000 --restart always -d caronc/apprise:latest

# 使用代理运行
docker run --name apprise \
  -p 8000:8000 \
  -e HTTP_PROXY=http://192.168.1.1:7890 \
  -e HTTPS_PROXY=http://192.168.1.1:7890 \
  --restart always \
  -d caronc/apprise:latest
```

## 📝 配置示例（config.yaml）

```yaml
# 配置通知渠道，将自动发送检测结果通知、新版本通知
# 复制 https://vercel.com/new/clone?repository-url=https://github.com/sinspired/apprise_vercel 到浏览器
# 按提示部署，建议为 Vercel 项目设置自定义域名（国内访问 Vercel 可能受限）。
# 填写搭建的 apprise API server 地址
# https://notify.xxxx.us.kg/notify
apprise-api-server: ""
# 通知渠道（支持 100+ 个渠道，格式请参照 https://github.com/caronc/apprise）
recipient-url:
  # - tgram://xxxxxx/-1002149239223
  # - dingtalk://xxxxxx@xxxxxxx
  # - mailto://xxxxx:xxxxxx@qq.com

# 自定义通知标题
notify-title: "🔔 节点状态更新"
```
