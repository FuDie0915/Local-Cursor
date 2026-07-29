package bridge

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	goruntime "runtime"

	"cursor/internal/appdata"
	"cursor/internal/cursor"
	"cursor/internal/logger"
	localruntime "cursor/internal/runtime"
)

// CLITerminalResult 返回打开 CLI 终端的结果。
type CLITerminalResult struct {
	Ok         bool   `json:"ok"`
	Error      string `json:"error"`
	WrapperDir string `json:"wrapperDir"`
}

// OpenCLITerminal 检测终端并打开一个预配置 wrapper 环境的终端窗口。
//
// 行为：
//   - 检测 CLI 版 cursor-agent 是否已安装
//   - 检测 wrapper 是否已生成（服务启动时自动生成）
//   - 以管理员/elevated 权限启动终端
//   - 终端启动时自动设置 wrapper 所需的环境变量
//   - 不自动执行 cursor-agent 命令（用户需自行 cd 到项目目录后调用）
func (s *WindowService) OpenCLITerminal() CLITerminalResult {
	cliPath := cursor.DetectCLIInstallPath()
	if cliPath == "" {
		return CLITerminalResult{
			Ok:    false,
			Error: "未检测到 CLI 版 cursor-agent，请先安装",
		}
	}

	if !cursor.CLIWrapperExists() {
		return CLITerminalResult{
			Ok:    false,
			Error: "CLI wrapper 尚未生成，请先启动服务",
		}
	}

	wrapperDir := cursor.CLIWrapperDir()
	caCertPath := filepath.Join(appdata.RootDir(), "cursor-local-ca.crt")

	if err := openElevatedTerminal(wrapperDir, caCertPath); err != nil {
		logger.Errorf("openCLITerminal failed: %v", err)
		return CLITerminalResult{
			Ok:         false,
			Error:      err.Error(),
			WrapperDir: wrapperDir,
		}
	}

	return CLITerminalResult{
		Ok:         true,
		WrapperDir: wrapperDir,
	}
}

// openElevatedTerminal 以管理员/elevated 权限启动终端，
// 并通过 shell 初始化脚本预设 wrapper 环境变量。
func openElevatedTerminal(workDir, caCertPath string) error {
	switch goruntime := detectOS(); goruntime {
	case "windows":
		return openWindowsTerminal(workDir, caCertPath)
	case "darwin":
		return openDarwinTerminal(workDir, caCertPath)
	default:
		return openLinuxTerminal(workDir, caCertPath)
	}
}

// ── Windows ──

func openWindowsTerminal(workDir, caCertPath string) error {
	// 生成临时 batch 文件设置环境变量，避免 PowerShell 内联引号转义问题
	batchPath, err := writeWindowsEnvBatch(workDir, caCertPath)
	if err != nil {
		return fmt.Errorf("生成终端初始化脚本失败: %w", err)
	}

	// 优先使用 Windows Terminal (wt.exe)，回退到 cmd.exe
	// batch 文件内部已 cd /d 到工作目录，无需 -WorkingDirectory 参数
	wtPath, wtErr := exec.LookPath("wt.exe")
	if wtErr == nil {
		// wt.exe: wt -d <dir> cmd.exe /k <batch>
		// 用数组传参避免引号嵌套
		args := []string{
			"-NoProfile", "-Command",
			fmt.Sprintf(
				"Start-Process -FilePath '%s' -ArgumentList @('-d','%s','cmd.exe','/k','%s') -Verb RunAs",
				wtPath, workDir, batchPath,
			),
		}
		return exec.Command("powershell", args...).Start()
	}

	// 回退到 cmd.exe，以管理员启动
	args := []string{
		"-NoProfile", "-Command",
		fmt.Sprintf(
			"Start-Process -FilePath 'cmd.exe' -ArgumentList @('/k','%s') -Verb RunAs",
			batchPath,
		),
	}
	return exec.Command("powershell", args...).Start()
}

// writeWindowsEnvBatch 生成临时 batch 文件设置 wrapper 环境变量。
func writeWindowsEnvBatch(workDir, caCertPath string) (string, error) {
	wrapperPath := cursor.CLIWrapperPath()
	port := extractPortFromWrapper(wrapperPath)
	preloadPath := filepath.Join(cursor.CLIWrapperDir(), "tls-bypass.cjs")
	authToken := localruntime.InjectAuthToken

	content := fmt.Sprintf(`@echo off
setlocal
set "HTTP_PROXY="
set "HTTPS_PROXY="
set "CLI_HTTPS_PORT=%s"
set "NODE_EXTRA_CA_CERTS=%s"
set "NODE_TLS_REJECT_UNAUTHORIZED=0"
set "NODE_OPTIONS=--require=%s"
set "CURSOR_AUTH_TOKEN=%s"
set "CURSOR_INVOKED_AS=agent.cmd"
cd /d "%s"
echo CLI environment initialized. cd to your project and run: cursor-agent
title Local-Cursor CLI Terminal
`, port, caCertPath, preloadPath, authToken, workDir)

	batchPath := filepath.Join(cursor.CLIWrapperDir(), "cli-terminal-init.bat")
	if err := os.WriteFile(batchPath, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("写入终端初始化脚本失败: %w", err)
	}
	return batchPath, nil
}

// extractPortFromWrapper 从 wrapper 脚本中提取 CLI_HTTPS_PORT 值。
func extractPortFromWrapper(wrapperPath string) string {
	data, err := os.ReadFile(wrapperPath)
	if err != nil {
		return "39092"
	}
	content := string(data)
	// 查找 set "CLI_HTTPS_PORT=xxxx" 或 export CLI_HTTPS_PORT="xxxx"
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "CLI_HTTPS_PORT") {
			// 提取 = 后面的值
			idx := strings.Index(line, "=")
			if idx >= 0 {
				val := strings.TrimSpace(line[idx+1:])
				val = strings.Trim(val, `"`)
				// 去掉 %...% 包裹
				val = strings.TrimPrefix(strings.TrimSuffix(val, "%"), "%")
				if val != "" && val != "CLI_HTTPS_PORT" {
					return val
				}
			}
		}
	}
	return "39092"
}

// ── macOS ──

func openDarwinTerminal(workDir, caCertPath string) error {
	// macOS: 用 osascript 以管理员权限打开 Terminal.app
	// 通过 AppleScript 执行 do script 预设环境变量
	preloadPath := filepath.Join(cursor.CLIWrapperDir(), "tls-bypass.cjs")
	port := extractPortFromWrapper(cursor.CLIWrapperPath())
	authToken := localruntime.InjectAuthToken

	// 构建初始化命令（不执行 cursor-agent）
	initCmd := fmt.Sprintf(
		`cd '%s' && unset HTTP_PROXY && unset HTTPS_PROXY && export CLI_HTTPS_PORT='%s' && export NODE_EXTRA_CA_CERTS='%s' && export NODE_TLS_REJECT_UNAUTHORIZED=0 && export NODE_OPTIONS='--require=%s' && export CURSOR_AUTH_TOKEN='%s' && echo 'CLI environment initialized. cd to your project and run: cursor-agent'`,
		workDir, port, caCertPath, preloadPath, authToken,
	)

	// 用 osascript 以管理员权限打开 Terminal 并执行初始化
	script := fmt.Sprintf(`
		do shell script "osascript -e 'tell application \"Terminal\" to do script \"%s\"' with administrator privileges" with prompt "Local-Cursor needs to open a terminal with administrator privileges"
	`, escapeAppleScript(initCmd))

	return exec.Command("osascript", "-e", script).Start()
}

// escapeAppleScript 转义 AppleScript 字符串中的特殊字符。
func escapeAppleScript(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

// ── Linux ──

func openLinuxTerminal(workDir, caCertPath string) error {
	// Linux: 尝试 pkexec + 常见终端模拟器
	// 检测可用的终端模拟器
	terminals := []string{"gnome-terminal", "konsole", "xterm", "xfce4-terminal", "alacritty", "kitty"}
	var terminalPath string
	for _, t := range terminals {
		if p, err := exec.LookPath(t); err == nil {
			terminalPath = p
			break
		}
	}
	if terminalPath == "" {
		return fmt.Errorf("未检测到可用的终端模拟器")
	}

	preloadPath := filepath.Join(cursor.CLIWrapperDir(), "tls-bypass.cjs")
	port := extractPortFromWrapper(cursor.CLIWrapperPath())
	authToken := localruntime.InjectAuthToken

	// 构建初始化命令
	initCmd := fmt.Sprintf(
		`cd '%s' && unset HTTP_PROXY && unset HTTPS_PROXY && export CLI_HTTPS_PORT='%s' && export NODE_EXTRA_CA_CERTS='%s' && export NODE_TLS_REJECT_UNAUTHORIZED=0 && export NODE_OPTIONS='--require=%s' && export CURSOR_AUTH_TOKEN='%s' && echo 'CLI environment initialized. cd to your project and run: cursor-agent' && exec bash`,
		workDir, port, caCertPath, preloadPath, authToken,
	)

	// 用 pkexec 以管理员权限启动终端
	// 不同终端的参数不同
	var cmd *exec.Cmd
	switch filepath.Base(terminalPath) {
	case "gnome-terminal":
		cmd = exec.Command("pkexec", terminalPath, "--", "bash", "-c", initCmd)
	case "konsole", "xfce4-terminal":
		cmd = exec.Command("pkexec", terminalPath, "-e", "bash", "-c", initCmd)
	case "xterm":
		cmd = exec.Command("pkexec", terminalPath, "-e", "bash", "-c", initCmd)
	case "alacritty":
		cmd = exec.Command("pkexec", terminalPath, "-e", "bash", "-c", initCmd)
	case "kitty":
		cmd = exec.Command("pkexec", terminalPath, "bash", "-c", initCmd)
	default:
		cmd = exec.Command("pkexec", terminalPath, "-e", "bash", "-c", initCmd)
	}
	return cmd.Start()
}

// detectOS 返回当前操作系统名称。
func detectOS() string {
	switch goruntime.GOOS {
	case "windows":
		return "windows"
	case "darwin":
		return "darwin"
	default:
		return "linux"
	}
}