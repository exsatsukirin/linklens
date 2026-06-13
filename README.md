# linklens

跨平台 Windows 快捷方式 (.lnk) 生成器 — 纯 Go，无需 Windows API。

Cross-platform Windows shortcut (.lnk) generator — pure Go, no Windows API required.

## 功能 / Features

- 生成 Windows `.lnk` 快捷方式文件 / Generate Windows `.lnk` shortcut files
- 纯 Go 实现，无 CGO，无 Windows API 调用 / Pure Go, no CGO, no Windows API
- 支持 Linux、macOS、Windows 跨平台 / Cross-platform (Linux, macOS, Windows)
- 支持非 ASCII 字符（中文、emoji 等）的 UTF-16LE 编码 / Non-ASCII character support (Chinese, emoji, etc.)

## 安装 / Install

```bash
go install github.com/exsatsukirin/linklens@latest
```

或从 [Releases](https://github.com/exsatsukirin/linklens/releases) 下载预编译二进制。

Or download a pre-built binary from [Releases](https://github.com/exsatsukirin/linklens/releases).

## 使用 / Usage

```bash
linklens create --target "C:\path\to\app.exe" --workdir "C:\path\to" --args "--verbose" --icon "C:\path\to\app.exe,0" output.lnk
```

### 参数 / Flags

| Flag | Short | 说明 / Description |
|------|-------|--------------------|
| `--target` | `-t` | 目标路径（必填）/ Target path (required) |
| `--workdir` | `-w` | 工作目录 / Working directory (Start in) |
| `--args` | `-a` | 命令行参数 / Command-line arguments |
| `--icon` | `-i` | 图标路径 / Icon file path |

### 示例 / Examples

```bash
# 最小用法 / Minimal
linklens create --target "C:\Windows\notepad.exe" notepad.lnk

# 完整参数 / Full options
linklens create \
  --target "C:\Program Files\MyApp\app.exe" \
  --workdir "C:\Program Files\MyApp" \
  --args "--config config.ini --verbose" \
  --icon "C:\Program Files\MyApp\app.exe,0" \
  myapp.lnk

# 中文路径 / Chinese path
linklens create \
  --target "C:\用户\文档\报告.docx" \
  --workdir "C:\用户\文档" \
  报告.lnk
```

## 技术实现 / Technical Details

直接实现 Microsoft Shell Link (.lnk) 二进制格式规范，完全不依赖 Windows API：

Implements the Microsoft Shell Link (.lnk) binary format directly, with zero Windows API calls:

| 块 / Block | 说明 / Description |
|------------|-------------------|
| **ShellLinkHeader** | 76 字节固定头 / 76-byte fixed header (magic + CLSID + flags) |
| **LinkTargetIDList** | 目标路径的 ITEMIDLIST / Target path ITEMIDLIST |
| **LinkInfo** | 卷信息 + 本地路径 / Volume info + local base path |
| **StringData** | 工作目录、参数、图标路径（UTF-16LE）/ Working dir, args, icon (UTF-16LE) |

## 构建 / Build

```bash
git clone https://github.com/exsatsukirin/linklens.git
cd linklens
go build -o linklens .
```

## 测试 / Tests

```bash
go test -race ./...
```

## 许可证 / License

MIT License. See [LICENSE](LICENSE) for details.
