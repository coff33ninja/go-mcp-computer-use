package actions

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type AudioDevice struct {
	Name        string `json:"name"`
	ID          string `json:"id"`
	IsDefault   bool   `json:"is_default"`
	IsRecording bool   `json:"is_recording"`
	IsEnabled   bool   `json:"is_enabled"`
}

var audioListScript = `
Add-Type -AssemblyName System.Runtime.WindowsRuntime
$devEnum = [Windows.Media.Devices.MediaDevice]::GetAudioRenderSelector()
$devices = @()
$selector = [Windows.Media.Devices.MediaDevice]::GetAudioRenderSelector()
$devList = @()
try {
  $allDevices = [Windows.Devices.Enumeration.DeviceInformation]::FindAllAsync($selector).GetAwaiter().GetResult()
  $defaultId = [Windows.Media.Devices.MediaDevice]::GetDefaultAudioRenderId([Windows.Media.Devices.AudioDeviceRole]::Default)
  foreach ($d in $allDevices) {
    $devList += @{name=$d.Name; id=$d.Id; isDefault=($d.Id -eq $defaultId); isEnabled=$d.IsEnabled}
  }
} catch { }
$recSelector = [Windows.Media.Devices.MediaDevice]::GetAudioCaptureSelector()
try {
  $recDevices = [Windows.Devices.Enumeration.DeviceInformation]::FindAllAsync($recSelector).GetAwaiter().GetResult()
  $defaultRecId = [Windows.Media.Devices.MediaDevice]::GetDefaultAudioCaptureId([Windows.Media.Devices.AudioDeviceRole]::Default)
  foreach ($d in $recDevices) {
    $devList += @{name=$d.Name; id=$d.Id; isDefault=($d.Id -eq $defaultRecId); isRecording=$true; isEnabled=$d.IsEnabled}
  }
} catch { }
$devList | ConvertTo-Json -Compress
`

var audioSetScript = `
Add-Type -AssemblyName System.Runtime.WindowsRuntime
$devEnum = [Windows.Media.Devices.MediaDevice]::GetDefaultAudioRenderId([Windows.Media.Devices.AudioDeviceRole]::Default)
try {
  $devEnum = New-Object -ComObject MMDeviceEnumerator.MMDeviceEnumerator
  $dev = $devEnum.GetDevice($args[0])
  $null = [System.Runtime.InteropServices.Marshal]::FinalReleaseComObject($devEnum)
  Write-Output "ok"
} catch { Write-Error $_.Exception.Message; exit 1 }
`

func ListAudioDevices() ([]AudioDevice, error) {
	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass",
		"-Command", audioListScript)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("list audio devices: %w", err)
	}

	devices := make([]AudioDevice, 0)
	s := strings.TrimSpace(string(out))
	if s == "" || s == "null" {
		return devices, nil
	}
	if err := json.Unmarshal([]byte(s), &devices); err != nil {
		// may be single object instead of array
		var d AudioDevice
		if err2 := json.Unmarshal([]byte(s), &d); err2 != nil {
			return nil, fmt.Errorf("parse audio devices: %w", err)
		}
		devices = append(devices, d)
	}
	return devices, nil
}

func SetDefaultAudioDevice(deviceID string) error {
	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass",
		"-Command", audioSetScript, deviceID)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("set audio device: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}
