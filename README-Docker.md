# 🌟 CfDDNS - Cloudflare Dynamic DNS Updater
[![](https://img.shields.io/github/v/release/aircross/cfddns.svg)](https://github.com/aircross/cfddns/releases)
[![GO Version](https://img.shields.io/github/go-mod/go-version/aircross/cfddns.svg)](#)
[![Downloads](https://img.shields.io/github/downloads/aircross/cfddns/total.svg)](#)
##### 这是一个设计用于利用CF的API TOKEN自动更新IP到Cloudflare,实现域名动态更行绑定的的小工具
##### 初衷是因为自己需要将自己家里的IPv6更新到CF，以便于可以快速地通过IPv6访问自己家里的电脑
##### 从一开始的网上找到尝试修改别人的，再到尝试用python手搓，到最后的golang手搓
##### 最终借助ChetGPT，以及到处搜到编写了本小工具
  
#### Docker使用方法
##### Docker快速部署
安装Docker
```
#国外服务器使用以下命令安装Docker
curl -fsSL https://get.docker.com | sh
# 设置开机自启
sudo systemctl enable docker.service
# 根据实际需要保留参数start|restart|stop
sudo service docker start|restart|stop
```

##### 运行Docker容器
```
mkdir -p /opt/docker/cfddns/
docker run --name cfddns -d --network host --restart=unless-stopped -v /opt/docker/cfddns/conf.toml:/usr/bin/cfddns/conf.toml  aircross/cfddns
```

🔑 CF_API_TOKEN是你的Cloudflare API token
  
`CF_API_TOKEN`应该是API **token** (_不是_ API key), 你可以在后面的链接处生成 [API Tokens页面](https://dash.cloudflare.com/profile/api-tokens). 通过 **Edit zone DNS** 模板来创建1个 token. 

#### 创建API Token教程如下：

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="./assets/images/api-tokens-1.png">
  <img alt="CF API Token 设置步骤1" src="./assets/images/api-tokens-1.png">
</picture>
<picture>
  <source media="(prefers-color-scheme: dark)" srcset="./assets/images/api-tokens-2.png">
  <img alt="CF API Token 设置步骤2" src="./assets/images/api-tokens-2.png">
</picture>
<picture>
  <source media="(prefers-color-scheme: dark)" srcset="./assets/images/api-tokens-3.png">
  <img alt="CF API Token 设置步骤3" src="./assets/images/api-tokens-3.png">
</picture>
<picture>
  <source media="(prefers-color-scheme: dark)" srcset="./assets/images/api-tokens-4.png">
  <img alt="CF API Token 设置步骤4" src="./assets/images/api-tokens-4.png">
</picture>


## 支持的架构和设备
<details>
  <summary>点击查看 支持的架构和设备</summary>

我们的平台提供与各种架构和设备的兼容性，确保在各种计算环境中的灵活性。以下是我们支持的关键架构：

- **amd64**: 这种流行的架构是个人计算机和服务器的标准，可以无缝地适应大多数现代操作系统。

- **x86 / i386**: 这种架构在台式机和笔记本电脑中被广泛采用，得到了众多操作系统和应用程序的广泛支持，包括但不限于 Windows、macOS 和 Linux 系统。

- **armv8 / arm64 / aarch64**: 这种架构专为智能手机和平板电脑等当代移动和嵌入式设备量身定制，以 Raspberry Pi 4、Raspberry Pi 3、Raspberry Pi Zero 2/Zero 2 W、Orange Pi 3 LTS 等设备为例。

- **armv7 / arm / arm32**: 作为较旧的移动和嵌入式设备的架构，它仍然广泛用于Orange Pi Zero LTS、Orange Pi PC Plus、Raspberry Pi 2等设备。
</details>

## 特别感谢

- [aircross](https://github.com/aircross/)
- [ChatGPT](https://chatgpt.com/)
- [Kimi.ai](https://kimi.moonshot.cn/)
- [Google](https://google.com/)
- [Golang](https://go.dev/)
- [百度](https://baidu.com/)

## 许可证

[](https://github.com/RandallAnjie/EmbyController#%E8%AE%B8%E5%8F%AF%E8%AF%81)

该项目使用Apache许可证。详情请参阅[LICENSE](https://github.com/RandallAnjie/EmbyController/blob/main/LICENSE)文件。
