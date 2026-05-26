# Legacy: PowerShell + WinForms 版本

这是 WinPHP 2025 的初始版本 (v1), 用纯 PowerShell + WinForms 实现.

**已被 [Go + Wails 版本](../README.md) 替代**, 仅保留作为参考与备份. 性能差, 不再维护.

## 启动 (如果你确实要用旧版)

```cmd
cd legacy
WinPHP.bat
```

(自动请求管理员权限. 调试用 `WinPHP-Console.bat`.)

## 已知问题

- 启动慢 (PowerShell 加载需要 2-3 秒)
- 状态轮询慢 (Get-Process 每次几百毫秒)
- 中文字符依赖 BOM, 容易出乱码
- WinForms 渲染老旧, 部分控件 (GroupBox 标题) 主题相关 bug

新版本 (Go + Wails) 已解决以上所有问题. 推荐使用根目录的新版.
