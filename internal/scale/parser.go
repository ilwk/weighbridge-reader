package scale

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	ModelDefault = "default"
	ModelHEBTW   = "heb-tw"
)

var weightPattern = regexp.MustCompile(`^\s*([+-]?)\s*(\d+)(\.\d+)?\s*kg\s*$`)

// NormalizeModel makes model selection insensitive to whitespace and casing.
func NormalizeModel(model string) string {
	model = strings.ToLower(strings.TrimSpace(model))
	if model == "" {
		return ModelDefault
	}
	return model
}

// Parse converts a model-specific frame to the stable WebSocket format.
func Parse(model, frame string) (string, error) {
	model = NormalizeModel(model)
	frame = strings.TrimSpace(frame)

	var payload string
	switch model {
	case ModelDefault:
		if !strings.HasPrefix(frame, "ST,GS") {
			return "", fmt.Errorf("default 型号报文应以 ST,GS 开头")
		}
		payload = strings.TrimPrefix(strings.TrimPrefix(frame, "ST,GS"), ",")
	case ModelHEBTW:
		if len(frame) < 2 || !strings.EqualFold(frame[:2], "wn") {
			return "", fmt.Errorf("heb-tw 型号报文应以 wn 开头")
		}
		payload = frame[2:]
	default:
		return "", fmt.Errorf("不支持的地磅型号 %q", model)
	}

	weight, err := normalizeWeight(payload)
	if err != nil {
		return "", fmt.Errorf("解析 %s 报文失败: %w", model, err)
	}
	return "ST,GS     " + weight + "kg", nil
}

func normalizeWeight(payload string) (string, error) {
	matches := weightPattern.FindStringSubmatch(payload)
	if matches == nil {
		return "", fmt.Errorf("重量格式无效: %q", payload)
	}

	integer := strings.TrimLeft(matches[2], "0")
	if integer == "" {
		integer = "0"
	}

	return matches[1] + integer + matches[3], nil
}
