package space

import "github.com/nilafzar/agents/services/manager/internal/model/provider"

type Space struct {
	ID         int64         `json:"id"`
	UserID     int64         `json:"userId"`
	Provider   provider.Kind `json:"provider"`
	Endpoint   string        `json:"endpoint"`
	RamCap     int32         `json:"ramCap"`
	CPUCap     int64         `json:"cpuCap"`
	StorageCap int64         `json:"storageCap"`
	UpTime     int64         `json:"uptime"`
	CreatedAt  int64         `json:"createdAt"`
	UpdatedAt  int64         `json:"updatedAt"`
}
