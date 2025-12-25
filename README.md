![BigLogo](./pic/ert_logo_big.png)

<h2 align="center" style="font-weight: 600">convertM3U8ToMP3</h2>
<p align="center">
    <s>更方便</s>的转换m3u8电台到mp3流
    <br />
    Version: 2025.12.25.1
    <!-- <a href="https://music.qier222.com" target="blank"><strong>🌎 访问DEMO</strong></a>  |  
    <a href="#%EF%B8%8F-安装" target="blank"><strong>📦️ 下载安装包</strong></a>  |  
    <a href="https://t.me/yesplaymusic" target="blank"><strong>💬 加入交流群</strong></a>
    <br />
    <br /> -->
</p>

## ✨ 特性

- ✅ 使用 Go 开发 性能至上
- 📃 支持主流的m3u8电台格式转换mp3流
- 🧩 感谢万能的ffmpeg
- 💾 修改后下载可自行修改config.json配置内容
- ✔️ 可以支持所有m3u8音频转换
- 📻 支持在欧卡2中使用
- 🖥️ 无UI界面 无感转换
- 📔 托盘程序 告别任务管理器

## 📦️ 打包

- 🛠 直接 git clone 然后安装依赖打包即可

```shell
# 安装goversioninfo以生成软件版本信息(可选 不要的话自行注释main.go第一行)
go install github.com/josephspurrier/goversioninfo/cmd/goversioninfo@latest

# 打包
go build

# Windows下打包如果想没有黑窗口 要加上 -ldflags "-H windowsgui"
go build -ldflags "-H windowsgui"
```

## 💻 运行

下载已构建的版本，双击运行，首次运行只会生成配置文件，再次运行即可使用。


## ☑️ Todo

1. 可以在软件中设置开机自启不必去`shell:Common Startup`

欢迎提 Issue 和 Pull request。

## 📜 开源许可

本项目仅供个人学习研究使用，禁止用于商业及非法用途。

基于 [GNU GPL v3](https://www.gnu.org/licenses/gpl-3.0.en.html#license-text) 许可进行开源。

## 🖼️ 截图

无Ui界面 无痛转换（确信）
