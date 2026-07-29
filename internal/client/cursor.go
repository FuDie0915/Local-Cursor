package client

import (
	"fmt"
	goruntime "runtime"

	"cursor/internal/cursor"
	"cursor/internal/logger"
)

// ApplyCursorSettings 用于处理与 ApplyCursorSettings 相关的逻辑。
func (s *ProxyService) ApplyCursorSettings() error {
	if s == nil || s.proxy == nil {
		return fmt.Errorf("proxy is not initialized")
	}
	s.caFileMu.Lock()
	caCertPath, err := cursor.EnsureCACertFile(s.caCertPEM, s.caFilePath)
	if err == nil {
		s.caFilePath = caCertPath
	}
	s.caFileMu.Unlock()
	if err != nil {
		return fmt.Errorf("ensure ca cert file: %w", err)
	}

	switch goruntime.GOOS {
	case "windows":
		if err := cursor.EnsureCACertInstalled(s.caCertPEM, caCertPath); err != nil {
			return fmt.Errorf("install ca cert: %w", err)
		}
	case "darwin":
		if err := cursor.EnsureCACertInstalled(s.caCertPEM, caCertPath); err != nil {
			return fmt.Errorf("install ca cert: %w", err)
		}
		if err := cursor.SetSystemNodeExtraCACerts(caCertPath); err != nil {
			return fmt.Errorf("set node extra ca certs: %w", err)
		}
	}

	proxyURL := cursor.ProxyURLFromListenAddr(s.proxy.Snapshot().ListenAddr)

	if err := cursor.WriteUserProxySettings(proxyURL); err != nil {
		return err
	}

	// CLI 版 wrapper: 不再需要 HTTP_PROXY，由 preload 脚本直接重定向 tls.connect 到后端 HTTPS 端口
	// 注入 NODE_EXTRA_CA_CERTS + CURSOR_AUTH_TOKEN
	// 如果未检测到 CLI 安装，跳过但不报错
	if cliPath := cursor.DetectCLIInstallPath(); cliPath != "" {
		cliHTTPSAddr := ""
		if s.backendHost != nil {
			cliHTTPSAddr = s.backendHost.CLIHTTPSListenAddr()
		}
		if err := cursor.WriteCLIProxyWrapper(cliHTTPSAddr, caCertPath); err != nil {
			logger.Errorf("write cli proxy wrapper failed: %v", err)
		}
		// 写入 mock auth.json，使 CLI 跳过交互式登录
		if err := cursor.WriteCLIAuthFile(); err != nil {
			logger.Errorf("write cli auth.json failed: %v", err)
		}
	} else {
		logger.Infof("applyCursorSettings: cli not detected, skipping wrapper")
	}

	s.setCursorSettingsApplied(true)
	return nil
}

// ClearCursorSettings 用于处理与 ClearCursorSettings 相关的逻辑。
func (s *ProxyService) ClearCursorSettings() error {
	if goruntime.GOOS == "darwin" {
		if err := cursor.ClearSystemNodeExtraCACerts(); err != nil {
			return err
		}
	}
	if err := cursor.ClearUserProxySettings(); err != nil {
		return err
	}
	if err := cursor.ClearCLIProxyWrapper(); err != nil {
		logger.Errorf("clear cli proxy wrapper failed: %v", err)
	}
	if err := cursor.ClearCLIAuthFile(); err != nil {
		logger.Errorf("clear cli auth.json failed: %v", err)
	}
	s.setCursorSettingsApplied(false)
	return nil
}

// GetDeviceID 用于处理与 GetDeviceID 相关的逻辑。
func (s *ProxyService) GetDeviceID() (string, error) {
	return cursor.GetDeviceID()
}

// CLIStatus 返回 CLI 版 cursor-agent 的代理状态。
type CLIStatus struct {
	Installed     bool   `json:"installed"`
	InstallPath   string `json:"installPath"`
	WrapperActive bool   `json:"wrapperActive"`
	WrapperPath   string `json:"wrapperPath"`
}

// GetCLIStatus 用于处理与 GetCLIStatus 相关的逻辑。
func (s *ProxyService) GetCLIStatus() CLIStatus {
	path := cursor.DetectCLIInstallPath()
	return CLIStatus{
		Installed:     path != "",
		InstallPath:   path,
		WrapperActive: cursor.CLIWrapperExists(),
		WrapperPath:   cursor.CLIWrapperPath(),
	}
}
