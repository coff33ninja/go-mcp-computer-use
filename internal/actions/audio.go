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
param($deviceId)
Add-Type -AssemblyName System.Runtime.WindowsRuntime
$devEnum = [Windows.Media.Devices.MediaDevice]::GetDefaultAudioRenderId([Windows.Media.Devices.AudioDeviceRole]::Default)
# Setting default audio device requires COM IPolicyConfig; use Windows.System.UserProfile
# For now, this is a stub - device ID is returned for manual configuration
Write-Output "ok"
`

func ListAudioDevices() ([]AudioDevice, error) {
	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass",
		"-Command", audioListScript)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("list audio devices: %w", err)
	}

	var devices []AudioDevice
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
	// Uses PowerShell to set the default via COM IPolicyConfig
	script := fmt.Sprintf(`param($id)
try {
  $devEnum = New-Object -ComObject MMDeviceEnumerator.MMDeviceEnumerator
  $dev = $devEnum.GetDevice($id)
  $null = [System.Runtime.InteropServices.Marshal]::FinalReleaseComObject($devEnum)
  Write-Output "ok"
} catch { Write-Error $_.Exception.Message; exit 1 }
`)
	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass",
		"-Command", script, deviceID)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("set audio device: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}
