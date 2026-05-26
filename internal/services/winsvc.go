//go:build windows

package services

import (
	"strings"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

// Windows 服务工具.

func ServiceExists(name string) bool {
	m, err := mgr.Connect()
	if err != nil {
		return false
	}
	defer m.Disconnect()
	s, err := m.OpenService(name)
	if err != nil {
		return false
	}
	_ = s.Close()
	return true
}

func ServiceStatusStr(name string) string {
	m, err := mgr.Connect()
	if err != nil {
		return ""
	}
	defer m.Disconnect()
	s, err := m.OpenService(name)
	if err != nil {
		return ""
	}
	defer s.Close()
	st, err := s.Query()
	if err != nil {
		return ""
	}
	switch st.State {
	case svc.Running:
		return "Running"
	case svc.Stopped:
		return "Stopped"
	case svc.StartPending:
		return "StartPending"
	case svc.StopPending:
		return "StopPending"
	case svc.Paused:
		return "Paused"
	}
	return ""
}

func StartService(name string) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	s, err := m.OpenService(name)
	if err != nil {
		return err
	}
	defer s.Close()
	return s.Start()
}

func StopService(name string) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	s, err := m.OpenService(name)
	if err != nil {
		return err
	}
	defer s.Close()
	st, err := s.Control(svc.Stop)
	if err != nil {
		// 已停止时 sc.Control 会报错, 视为成功
		if strings.Contains(err.Error(), "1062") || strings.Contains(err.Error(), "not started") {
			return nil
		}
		return err
	}
	// 等待停止
	deadline := time.Now().Add(15 * time.Second)
	for st.State != svc.Stopped && time.Now().Before(deadline) {
		time.Sleep(200 * time.Millisecond)
		st, err = s.Query()
		if err != nil {
			break
		}
	}
	return nil
}
