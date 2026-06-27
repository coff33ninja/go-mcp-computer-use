package actions

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

func SetBrightness(percent int) error {
	if percent < 0 || percent > 100 {
		return fmt.Errorf("brightness must be 0-100")
	}
	cmd := exec.Command("powershell", "-NoProfile", "-Command",
		fmt.Sprintf(`(Get-WmiObject -Namespace root/wmi -Class WmiMonitorBrightnessMethods).WmiSetBrightness(1,%d)`, percent))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("set brightness: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

func GetBrightness() (int, error) {
	cmd := exec.Command("powershell", "-NoProfile", "-Command",
		`(Get-WmiObject -Namespace root/wmi -Class WmiMonitorBrightness).CurrentBrightness`)
	out, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("get brightness: %w", err)
	}
	s := strings.TrimSpace(string(out))
	if s == "" {
		return 0, fmt.Errorf("brightness not supported on this display")
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("parse brightness: %w", err)
	}
	return v, nil
}
