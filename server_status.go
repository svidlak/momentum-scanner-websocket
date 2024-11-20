package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

type ServerStatus struct {
	CPUUsage         string          `json:"cpu_usage"`
	MemUsed          string          `json:"mem_used"`
	MemTotal         string          `json:"mem_total"`
	LastMessage      json.RawMessage `json:"last_message"`
	ConnectedClients int             `json:"connected_clients"`
	Alive            bool            `json:"alive"`
}

func serverStatusHandler(w http.ResponseWriter, r *http.Request) {
	cpuPercent, _ := cpu.Percent(0, false)
	memStats, _ := mem.VirtualMemory()

	cpuUsage := fmt.Sprintf("%.2f%%", cpuPercent[0])
	memUsed := formatMemory(memStats.Used)
	memTotal := formatMemory(memStats.Total)

	status := ServerStatus{
		CPUUsage:         cpuUsage,
		MemUsed:          memUsed,
		MemTotal:         memTotal,
		Alive:            true,
		LastMessage:      lastWebSocketMessage,
		ConnectedClients: len(clients),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(status)
}

func formatMemory(bytes uint64) string {
	if bytes >= 1<<30 {
		return fmt.Sprintf("%.2f GB", float64(bytes)/(1<<30))
	} else if bytes >= 1<<20 {
		return fmt.Sprintf("%.2f MB", float64(bytes)/(1<<20))
	}
	return fmt.Sprintf("%d KB", bytes/1024)
}
