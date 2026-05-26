package services

// 通用类型与接口

type Status struct {
	Running          bool   `json:"running"`
	Version          string `json:"version"`
	Port             int    `json:"port"`
	ServiceInstalled bool   `json:"serviceInstalled"`
	ServiceStatus    string `json:"serviceStatus"` // Running/Stopped/...
	BinInstalled     bool   `json:"binInstalled"`
}

type Service interface {
	Name() string
	Start() error
	Stop() error
	Restart() error
	Status() Status
}
