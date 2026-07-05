# VibeArk

便携应用管理器 — 一行命令下载、安装、更新常用 Windows 便携软件。

Portable App Manager — download, install, and update common Windows portable apps with one keystroke.

> **仅支持 Windows x64 / Windows x64 only** | 针对中国网络环境适配（代理、镜像站、GitHub 加速）/ Optimized for China's network environment (proxy, mirrors, GitHub acceleration).

---

## 支持的软件 / Supported Apps

| 软件 / App | 描述 / Description | 类型 / Type |
|---|---|---|
| Ungoogled Chromium | 去 Google 化的 Chromium 浏览器 | 浏览器 / Browser |
| TreeSize Free | 磁盘空间分析工具 | 系统 / System |
| Geek Uninstaller | 轻量卸载工具 | 系统 / System |
| ScreenToGif | 屏幕录制转 GIF 工具 | 媒体 / Media |
| Rufus | USB 启动盘制作工具 | 系统 / System |
| PeaZip | 开源压缩工具 | 系统 / System |
| imFile | 下载管理器 | 网络 / Network |
| ShareX | 屏幕截图与录制工具 | 媒体 / Media |
| v2rayN | 代理客户端 | 网络 / Network |
| Notepad3 | 轻量文本编辑器 | 编辑 / Editor |
| qView | 轻量图片浏览器 | 媒体 / Media |
| DiskGenius | 硬盘分区与数据恢复软件 | 系统 / System |
| VLC Media Player | 开源多媒体播放器 | 媒体 / Media |
| FileZilla | 开源 FTP/FTPS/SFTP 客户端 | 网络 / Network |

---

## 功能 / Features

- **全键盘操作 / Keyboard-driven**: 无需鼠标，所有操作通过快捷键完成
- **多源下载 / Multi-source**: 支持官方网站下载、GitHub Releases、镜像站加速、CDN 直链、Chocolatey API
- **版本检查 / Version check**: 逐个检查或一键检查全部软件的最新版本（Ctrl+U）
- **代理支持 / Proxy support**: 内置 HTTP 代理端口配置，一键开关
- **GitHub 加速 / GitHub proxy**: 可配置 GitHub 下载代理前缀（如 gh-proxy.org）
- **自动解压 / Auto-extract**: 下载后自动解压 ZIP/7z/exe，智能扁平化目录结构
- **安装追踪 / Install tracking**: 本地 lock 文件记录安装版本、exe 路径、时间戳
- **中英双语 / i18n**: Ctrl+L 一键切换中文/英文，自动检测系统语言
- **自定义安装目录 / Custom install dir**: 默认 `D:/ArkApps`，可在设置中修改
- **实时进度 / Real-time progress**: 下载进度百分比、文件大小实时显示

---

## 安装 / Installation

### 方式一：下载预编译版本（推荐）

从 [Releases](https://github.com/elwina/VibeArk/releases) 下载 `vibeark.exe`，放到任意目录即可运行。

### 方式二：从源码构建

```bash
git clone git@github.com:elwina/VibeArk.git
cd VibeArk
go build -o dist/vibeark.exe .
```

构建时注入版本号：

```bash
go build -ldflags "-X vibeark/internal/version.Value=1.0.0" -o dist/vibeark.exe .
```

---

## 使用 / Usage

启动程序：

```bash
./vibeark.exe
```

### 主界面快捷键 / Main View Keys

| 按键 / Key | 功能 / Action |
|---|---|
| `w` / `↑` | 上移光标 / Move up |
| `s` / `↓` | 下移光标 / Move down |
| `Enter` / `d` | 下载安装 / Download & install |
| `o` | 启动已安装软件 / Launch installed app |
| `a` / `u` | 检查当前软件更新 / Check update |
| `Ctrl+U` | 检查全部软件更新 / Check all updates |
| `x` | 删除已安装软件 / Delete installed app |
| `p` | 开关代理 / Toggle proxy |
| `g` | 开关 GitHub 加速 / Toggle GitHub proxy |
| `r` | 刷新列表 / Refresh list |
| `Ctrl+L` | 切换语言 / Toggle language (中文 / English) |
| `Ctrl+P` | 打开设置 / Open settings |
| `Esc` | 取消下载 / Cancel download |
| `q` | 退出 / Quit |

### 设置界面 / Settings View

| 字段 / Field | 说明 / Description |
|---|---|
| 安装目录 / Install Dir | 软件安装的根目录，默认 `D:/ArkApps` |
| 代理端口 / Proxy Port | HTTP 代理端口号，如 `10808` |
| GitHub加速 / GitHub Proxy | GitHub 下载加速前缀，如 `https://gh-proxy.org/` |

设置保存在 `~/.vibeark/settings.yaml`，安装记录保存在 `~/.vibeark/lock.yaml`。

---

## 目录结构 / Directory Structure

```
VibeArk/
├── main.go                  # 入口 / Entry point
├── cmd/                     # CLI 命令（list / install / check）
│   ├── root.go
│   └── tui.go
├── tui/                     # TUI 界面（Bubble Tea）
│   ├── app.go               # 主界面 / Main view
│   └── app_test.go          # 单元测试 / Unit tests
├── internal/
│   ├── apps/                # 软件定义
│   │   ├── definitions.go   # 14 款软件配置
│   │   └── types.go         # 数据结构
│   ├── config/              # 设置 & 锁文件
│   │   ├── settings.go      # 设置读写
│   │   └── lock.go          # 安装记录
│   ├── downloader/          # 下载 & 解压
│   │   └── downloader.go    # 多源下载、解压、扁平化
│   ├── httpclient/          # HTTP 客户端
│   │   ├── client.go        # 带超时/代理的 HTTP 客户端
│   │   └── lanzou.go        # 蓝奏云链接解析器（备用）
│   ├── i18n/                # 国际化
│   │   └── i18n.go          # 64 条翻译条目 + 系统语言检测
│   ├── scanner/             # 本地扫描
│   │   └── scanner.go       # 扫描已安装软件及版本
│   ├── updater/             # 版本检查
│   │   ├── updater.go       # 多源版本检查（web-scrape / GitHub）
│   │   └── version.go       # 版本号比较
│   └── version/             # 程序版本
│       └── version.go       # 编译时注入或默认 0.1.0
├── dist/                    # 构建产物（git 忽略）
├── .gitignore
└── go.mod / go.sum
```

---

## 技术架构 / Architecture

```
┌──────────────────────────────────────────┐
│                   TUI                     │
│  Bubble Tea + lipgloss                   │
│  Model -> Update -> View 事件循环         │
│  channel 驱动的异步下载进度               │
└───────────┬──────────────────────────────┘
            │
    ┌───────┼────────┐
    ▼       ▼        ▼
  scanner  updater  downloader
          (web-scrape / GitHub API)
                       │
                       ▼
                  httpclient
            (代理 / Cookie Jar / 蓝奏解析)
```

- **版本检查**: web-scrape（正则提取页面版本号）或 GitHub Releases API
- **下载**: 支持直链、GitHub Releases、redirect-page（从跳转页提取真实 URL）、蓝奏云解析
- **解压**: PowerShell `Expand-Archive`（ZIP）或 7z 命令行（7z），自动扁平化嵌套目录，清理 `__MACOSX`
- **安装追踪**: YAML lock 文件记录每个软件的安装状态、版本、exe 路径、时间戳

---

## 添加新软件 / Adding a New App

在 `internal/apps/definitions.go` 的 `AppDefinitions` 数组中添加一个 `AppEntry`，需要填写以下字段：

```go
{
    Name:          "软件名称",
    ID:            "unique-id",          // 唯一标识，用作安装目录名
    Description:   "中文描述",
    DescriptionEn: "English description",
    Source: AppSource{
        Type:           "web-scrape",    // "web-scrape" | "github" | "direct-download"
        URL:            "https://...",   // 版本检查页面地址
        VersionPattern: `regex`,         // 正则提取版本号
    },
    Download: AppDownload{
        Type: "direct",                  // "direct" | "github-release" | "redirect-page"
        URL:  "https://.../{version}...", // {version} 会被替换为实际版本号
        UA:   "",                        // 可选，自定义 User-Agent
    },
    ExePath: "app.exe",                  // 主程序名（支持 * 通配符）
    Tags:    []string{"category"},
},
```

**Source.Type 说明**:

| 类型 | 说明 |
|---|---|
| `web-scrape` | 抓取页面 HTML，用正则提取版本号 |
| `github` | 通过 GitHub Releases API 获取最新版本和下载链接 |
| `direct-download` | 不检查版本，直接下载（适合固定 URL） |

**Download.Type 说明**:

| 类型 | 说明 |
|---|---|
| `direct` | 直接下载，URL 中的 `{version}` 会被替换 |
| `github-release` | 从 GitHub Releases 的 assets 中匹配下载文件 |
| `redirect-page` | 先请求跳转页面，用 `RedirectURLPattern` 正则提取真实下载 URL |

---

## 许可证 / License

MIT

---

## 作者 / Author

[Elwina Vardal](https://github.com/elwina)
