package configurator

import (
	"context"
	"encoding/json"
	"log/slog"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// HardwareProfile captures machine hardware capabilities relevant to model selection.
type HardwareProfile struct {
	TotalRAMMB int64  `json:"total_ram_mb"`
	CPUCores   int    `json:"cpu_cores"`
	Arch       string `json:"arch"`        // "arm64" | "amd64"
	GPUVendor  string `json:"gpu_vendor"`  // "apple_metal" | "nvidia" | "amd" | "none" | "unknown"
	GPUVRAM_MB int64  `json:"gpu_vram_mb"` // 0 if no dedicated GPU or unknown
}

// Detect collects hardware information from the current machine.
// On any detection failure the field is set to zero/unknown — startup is never blocked.
func Detect() HardwareProfile {
	hw := HardwareProfile{
		Arch:     runtime.GOARCH,
		CPUCores: runtime.NumCPU(),
	}
	hw.TotalRAMMB = detectRAM()
	hw.GPUVendor, hw.GPUVRAM_MB = detectGPU()
	return hw
}

func detectRAM() int64 {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "sysctl", "-n", "hw.memsize").Output()
	if err != nil {
		slog.Warn("hardware detection failed", "field", "total_ram_mb", "err", err)
		return 0
	}
	bytes, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	if err != nil {
		slog.Warn("hardware detection parse failed", "field", "total_ram_mb", "err", err)
		return 0
	}
	return bytes / 1048576
}

func detectGPU() (vendor string, vramMB int64) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "system_profiler", "SPDisplaysDataType", "-json").Output()
	if err != nil {
		slog.Warn("hardware detection failed", "field", "gpu", "err", err)
		return "unknown", 0
	}
	return parseGPUInfo(out)
}

func parseGPUInfo(data []byte) (vendor string, vramMB int64) {
	var result struct {
		SPDisplaysDataType []map[string]interface{} `json:"SPDisplaysDataType"`
	}
	if err := json.Unmarshal(data, &result); err != nil || len(result.SPDisplaysDataType) == 0 {
		return "unknown", 0
	}
	display := result.SPDisplaysDataType[0]

	// Vendor
	if v, ok := display["spdisplays_vendor"].(string); ok {
		switch {
		case strings.Contains(v, "Apple"):
			vendor = "apple_metal"
		case strings.Contains(v, "NVIDIA"):
			vendor = "nvidia"
		case strings.Contains(v, "AMD"):
			vendor = "amd"
		default:
			vendor = "none"
		}
	} else {
		vendor = "none"
	}

	// VRAM
	for _, key := range []string{"spdisplays_vram", "spdisplays_vram_shared"} {
		if raw, ok := display[key].(string); ok {
			fields := strings.Fields(raw)
			if len(fields) >= 2 {
				val, err := strconv.ParseInt(fields[0], 10, 64)
				if err == nil {
					unit := strings.ToUpper(fields[1])
					if unit == "GB" {
						val *= 1024
					}
					vramMB = val
					break
				}
			}
		}
	}
	return vendor, vramMB
}
