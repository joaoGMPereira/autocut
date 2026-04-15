package configurator

import "fmt"

// ModelRecommendation holds AI model recommendations based on hardware.
type ModelRecommendation struct {
	WhisperTier      string `json:"whisper_tier"`
	WhisperReason    string `json:"whisper_reason"`
	WhisperSizeMB    int    `json:"whisper_size_mb"`
	OllamaModel      string `json:"ollama_model"`
	OllamaReason     string `json:"ollama_reason"`
	UpgradeAvailable bool   `json:"upgrade_available"`
}

type whisperTierInfo struct {
	name   string
	sizeMB int
}

var whisperTiers = []whisperTierInfo{
	{"tiny", 39},
	{"base", 74},
	{"small", 244},
	{"medium", 769},
	{"large", 1550},
}

func hasGPU(hw HardwareProfile) bool {
	return hw.GPUVendor == "apple_metal" || hw.GPUVendor == "nvidia"
}

// Recommend computes model recommendations from detected hardware.
func Recommend(hw HardwareProfile) ModelRecommendation {
	return ModelRecommendation{
		WhisperTier:      whisperTier(hw),
		WhisperReason:    whisperReason(hw),
		WhisperSizeMB:    whisperSizeMB(hw),
		OllamaModel:      ollamaModel(hw),
		OllamaReason:     ollamaReason(hw),
		UpgradeAvailable: upgradeAvailable(hw),
	}
}

func whisperTier(hw HardwareProfile) string {
	gpu := hasGPU(hw)
	ram := hw.TotalRAMMB
	switch {
	case ram < 4*1024:
		if gpu {
			return "base"
		}
		return "tiny"
	case ram < 8*1024:
		if gpu {
			return "small"
		}
		return "base"
	case ram < 16*1024:
		if gpu {
			return "medium"
		}
		return "small"
	default:
		if gpu {
			return "large"
		}
		return "medium"
	}
}

func whisperSizeMB(hw HardwareProfile) int {
	tier := whisperTier(hw)
	for _, t := range whisperTiers {
		if t.name == tier {
			return t.sizeMB
		}
	}
	return 0
}

func whisperReason(hw HardwareProfile) string {
	ramGB := hw.TotalRAMMB / 1024
	tier := whisperTier(hw)
	sizeMB := whisperSizeMB(hw)
	gpu := "no GPU"
	if hasGPU(hw) {
		gpu = hw.GPUVendor + " GPU"
	}
	return fmt.Sprintf("%d GB RAM with %s — %s model (%d MB) recommended", ramGB, gpu, tier, sizeMB)
}

func ollamaModel(hw HardwareProfile) string {
	switch {
	case hw.TotalRAMMB < 8*1024:
		return "llama3.2:1b"
	case hw.TotalRAMMB < 16*1024:
		return "llama3.2:3b"
	default:
		return "llama3.1:8b"
	}
}

func ollamaReason(hw HardwareProfile) string {
	ramGB := hw.TotalRAMMB / 1024
	model := ollamaModel(hw)
	var sizeNote string
	switch model {
	case "llama3.2:1b":
		sizeNote = "~0.9 GB"
	case "llama3.2:3b":
		sizeNote = "~2 GB"
	default:
		sizeNote = "~5 GB"
	}
	return fmt.Sprintf("%d GB RAM — %s (%s) fits comfortably", ramGB, model, sizeNote)
}

func upgradeAvailable(hw HardwareProfile) bool {
	currentTier := whisperTier(hw)
	currentIdx := -1
	for i, t := range whisperTiers {
		if t.name == currentTier {
			currentIdx = i
			break
		}
	}
	if currentIdx < 0 || currentIdx >= len(whisperTiers)-1 {
		return false
	}
	nextTier := whisperTiers[currentIdx+1]
	// Upgrade available if next tier size fits in 50% of RAM
	halfRAM := hw.TotalRAMMB / 2
	return int64(nextTier.sizeMB) <= halfRAM
}
