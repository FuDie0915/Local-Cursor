package bridge

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	goruntime "runtime"

	"cursor/internal/cursor"
	"cursor/internal/logger"
)

// CLITerminalResult 返回打开 CLI 终端的结果。
type CLITerminalResult struct {
	Ok         bool   `json:"ok"`
	Error      string `json:"error"`
	WrapperDir string `json:"wrapperDir"`
}

// cliTerminalInitFilename 是 Windows 终端初始化批处理脚本的文件名。
const cliTerminalInitFilename = "cli-terminal-init.bat"

// extractEnvSetupLines 从 wrapper 脚本提取环境变量设置行。
// wrapper 是 CLI 环境配置的唯一来源，终端初始化脚本从中派生，
// 避免两套独立的环境变量配置逻辑产生不一致。
func extractEnvSetupLines(wrapperPath string) ([]string, error) {
	data, err := os.ReadFile(wrapperPath)
	if err != nil {
		return nil, err
	}
	var lines []string
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "set \"") || // Windows: set "KEY=VALUE"
			strings.HasPrefix(trimmed, "export ") || // Unix: export KEY="VALUE"
			strings.HasPrefix(trimmed, "unset ") { // Unix: unset KEY
			lines = append(lines, trimmed)
		}
	}
	return lines, nil
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

	if err := openElevatedTerminal(wrapperDir); err != nil {
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
func openElevatedTerminal(workDir string) error {
	switch detectOS() {
	case "windows":
		return openWindowsTerminal(workDir)
	case "darwin":
		return openDarwinTerminal(workDir)
	default:
		return openLinuxTerminal(workDir)
	}
}

// ── Windows ──

func openWindowsTerminal(workDir string) error {
	// 生成 batch 文件设置环境变量，避免 PowerShell 内联引号转义问题
	batchPath, err := writeWindowsEnvBatch(workDir)
	if err != nil {
		return fmt.Errorf("生成终端初始化脚本失败: %w", err)
	}

	// 优先使用 Windows Terminal (wt.exe)，回退到 cmd.exe
	// batch 文件内部已 cd /d 到工作目录，无需 -WorkingDirectory 参数
	wtPath, wtErr := exec.LookPath("wt.exe")
	if wtErr == nil {
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

// writeWindowsEnvBatch 生成 batch 文件设置 wrapper 环境变量。
// 环境变量配置直接从 wrapper 脚本提取，确保与 wrapper 保持一致。
func writeWindowsEnvBatch(workDir string) (string, error) {
	envLines, err := extractEnvSetupLines(cursor.CLIWrapperPath())
	if err != nil {
		return "", fmt.Errorf("读取 wrapper 环境配置失败: %w", err)
	}

	var b strings.Builder
	b.WriteString("@echo off\n")
	for _, line := range envLines {
		b.WriteString(line)
		b.WriteByte('\n')
	}
	fmt.Fprintf(&b, "cd /d \"%s\"\n", workDir)
	b.WriteString("echo CLI environment initialized. cd to your project and run: cursor-agent\n")
	b.WriteString("title Local-Cursor CLI Terminal\n")

	batchPath := filepath.Join(cursor.CLIWrapperDir(), cliTerminalInitFilename)
	if err := os.WriteFile(batchPath, []byte(b.String()), 0o644); err != nil {
		return "", fmt.Errorf("写入终端初始化脚本失败: %w", err)
	}
	return batchPath, nil
}

// ── macOS ──

func openDarwinTerminal(workDir string) error {
	envLines, err := extractEnvSetupLines(cursor.CLIWrapperPath())
	if err != nil {
		return fmt.Errorf("读取 wrapper 环境配置失败: %w", err)
	}

	parts := []string{fmt.Sprintf("cd '%s'", workDir)}
	parts = append(parts, envLines...)
	parts = append(parts, "echo 'CLI environment initialized. cd to your project and run: cursor-agent'")
	initCmd := strings.Join(parts, " && ")

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

func openLinuxTerminal(workDir string) error {
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

	envLines, err := extractEnvSetupLines(cursor.CLIWrapperPath())
	if err != nil {
		return fmt.Errorf("读取 wrapper 环境配置失败: %w", err)
	}

	parts := []string{fmt.Sprintf("cd '%s'", workDir)}
	parts = append(parts, envLines...)
	parts = append(parts, "echo 'CLI environment initialized. cd to your project and run: cursor-agent'")
	parts = append(parts, "exec bash")
	initCmd := strings.Join(parts, " && ")

	// 用 pkexec 以管理员权限启动终端，不同终端的参数不同
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