//go:build !windows

package services

func ServiceExists(name string) bool     { return false }
func ServiceStatusStr(name string) string { return "" }
func StartService(name string) error      { return nil }
func StopService(name string) error       { return nil }
