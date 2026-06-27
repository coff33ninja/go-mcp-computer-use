package actions

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type UIAElement struct {
	Name          string  `json:"name"`
	AutomationID  string  `json:"automation_id,omitempty"`
	ControlType   string  `json:"control_type,omitempty"`
	IsEnabled     bool    `json:"is_enabled"`
	X             float64 `json:"x"`
	Y             float64 `json:"y"`
	Width         float64 `json:"width"`
	Height        float64 `json:"height"`
	ProcessID     int     `json:"process_id,omitempty"`
}

type UIAFindOpts struct {
	Name          string `json:"name,omitempty"`
	AutomationID  string `json:"automation_id,omitempty"`
	ControlType   string `json:"control_type,omitempty"`
	WaitMs        int    `json:"wait_ms,omitempty"`
}

var uiaFindScript = `param($paramsJson)
Add-Type -AssemblyName UIAutomationClient
Add-Type -AssemblyName UIAutomationTypes
$p = $paramsJson | ConvertFrom-Json
$conds = @()
if ($p.name) { $conds += [System.Windows.Automation.PropertyCondition]::new([System.Windows.Automation.AutomationElement]::NameProperty, $p.name) }
if ($p.automation_id) { $conds += [System.Windows.Automation.PropertyCondition]::new([System.Windows.Automation.AutomationElement]::AutomationIdProperty, $p.automation_id) }
if ($p.control_type) {
  $ct = [System.Windows.Automation.ControlType]::$($p.control_type)
  $conds += [System.Windows.Automation.PropertyCondition]::new([System.Windows.Automation.AutomationElement]::ControlTypeProperty, $ct)
}
$cond = $null
if ($conds.Count -eq 0) { Write-Output '[]'; exit }
if ($conds.Count -eq 1) { $cond = $conds[0] } else { $cond = [System.Windows.Automation.AndCondition]::new($conds) }
$root = [System.Windows.Automation.AutomationElement]::RootElement
$results = $root.FindAll([System.Windows.Automation.TreeScope]::Descendants, $cond)
if ($results.Count -eq 0) {
  $results = $root.FindAll([System.Windows.Automation.TreeScope]::Children, $cond)
}
$out = @()
foreach ($elem in $results) {
  $r = $elem.Current.BoundingRectangle
  $out += @{
    name = $elem.Current.Name
    automation_id = $elem.Current.AutomationId
    control_type = $elem.Current.ControlType.ProgrammaticName
    is_enabled = $elem.Current.IsEnabled
    x = [double]$r.X
    y = [double]$r.Y
    width = [double]$r.Width
    height = [double]$r.Height
    process_id = $elem.Current.ProcessId
  }
}
$out | ConvertTo-Json -Compress
`

var uiaGetTextScript = `param($paramsJson)
Add-Type -AssemblyName UIAutomationClient
Add-Type -AssemblyName UIAutomationTypes
$p = $paramsJson | ConvertFrom-Json
$root = [System.Windows.Automation.AutomationElement]::RootElement
$cond = $null
if ($p.automation_id) {
  $cond = [System.Windows.Automation.PropertyCondition]::new([System.Windows.Automation.AutomationElement]::AutomationIdProperty, $p.automation_id)
} elseif ($p.name) {
  $cond = [System.Windows.Automation.PropertyCondition]::new([System.Windows.Automation.AutomationElement]::NameProperty, $p.name)
} else { Write-Output '{"text":""}'; exit }
$elem = $root.FindFirst([System.Windows.Automation.TreeScope]::Descendants, $cond)
if ($elem -eq $null) { Write-Output '{"text":""}'; exit }
try {
  $vp = $elem.GetCurrentPattern([System.Windows.Automation.ValuePattern]::Pattern)
  $text = $vp.Current.Value
  $out = @{text = $text}
  $out | ConvertTo-Json -Compress
} catch {
  try {
    $tp = $elem.GetCurrentPattern([System.Windows.Automation.TextPattern]::Pattern)
    $text = $tp.DocumentRange.GetText(65536)
    $out = @{text = $text}
    $out | ConvertTo-Json -Compress
  } catch {
    Write-Output '{"text":""}'
  }
}
`

var uiaInvokeScript = `param($paramsJson)
Add-Type -AssemblyName UIAutomationClient
Add-Type -AssemblyName UIAutomationTypes
$p = $paramsJson | ConvertFrom-Json
$root = [System.Windows.Automation.AutomationElement]::RootElement
$conds = @()
if ($p.name) { $conds += [System.Windows.Automation.PropertyCondition]::new([System.Windows.Automation.AutomationElement]::NameProperty, $p.name) }
if ($p.automation_id) { $conds += [System.Windows.Automation.PropertyCondition]::new([System.Windows.Automation.AutomationElement]::AutomationIdProperty, $p.automation_id) }
$cond = $null
if ($conds.Count -eq 0) { Write-Output 'false'; exit }
if ($conds.Count -eq 1) { $cond = $conds[0] } else { $cond = [System.Windows.Automation.AndCondition]::new($conds) }
$elem = $root.FindFirst([System.Windows.Automation.TreeScope]::Descendants, $cond)
if ($elem -eq $null) { Write-Output 'false'; exit }
try {
  $invoke = $elem.GetCurrentPattern([System.Windows.Automation.InvokePattern]::Pattern)
  $invoke.Invoke()
  Write-Output 'true'
} catch {
  try {
    $toggle = $elem.GetCurrentPattern([System.Windows.Automation.TogglePattern]::Pattern)
    $toggle.Toggle()
    Write-Output 'true'
  } catch {
    try {
      $click = $elem.GetCurrentPattern([System.Windows.Automation.ClickPattern]::Pattern)
      $click.Click()
      Write-Output 'true'
    } catch {
      Write-Output 'false'
    }
  }
}
`

type UIATextResult struct {
	Text string `json:"text"`
}

func runUIAScript(script, paramsJSON string, result interface{}) error {
	tmpDir := os.Getenv("TEMP")
	if tmpDir == "" {
		tmpDir = os.TempDir()
	}
	scriptPath := filepath.Join(tmpDir, fmt.Sprintf("uia_%d.ps1", time.Now().UnixNano()))
	scriptContent := script + " " + paramsJSON
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0); err != nil {
		return fmt.Errorf("uia write script: %w", err)
	}
	defer os.Remove(scriptPath)

	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", scriptPath)
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("uia exec: %w (stderr: %s)", err, string(ee.Stderr))
		}
		return fmt.Errorf("uia exec: %w", err)
	}

	s := string(out)
	if s == "" {
		return nil
	}

	if err := json.Unmarshal([]byte(s), result); err != nil {
		return fmt.Errorf("uia parse: %w", err)
	}
	return nil
}

func UIAFindElement(opts UIAFindOpts) ([]UIAElement, error) {
	if opts.Name == "" && opts.AutomationID == "" && opts.ControlType == "" {
		return nil, fmt.Errorf("uia_find: at least one of name, automation_id, control_type required")
	}

	data, _ := json.Marshal(opts)
	var elements []UIAElement
	if err := runUIAScript(uiaFindScript, string(data), &elements); err != nil {
		return nil, fmt.Errorf("uia_find: %w", err)
	}
	return elements, nil
}

func UIAGetText(name, automationID string) (string, error) {
	if name == "" && automationID == "" {
		return "", fmt.Errorf("uia_get_text: name or automation_id required")
	}

	opts := map[string]string{}
	if name != "" {
		opts["name"] = name
	}
	if automationID != "" {
		opts["automation_id"] = automationID
	}
	data, _ := json.Marshal(opts)
	var result UIATextResult
	if err := runUIAScript(uiaGetTextScript, string(data), &result); err != nil {
		return "", fmt.Errorf("uia_get_text: %w", err)
	}
	return result.Text, nil
}

func UIAInvoke(name, automationID string) (bool, error) {
	if name == "" && automationID == "" {
		return false, fmt.Errorf("uia_invoke: name or automation_id required")
	}

	opts := map[string]string{}
	if name != "" {
		opts["name"] = name
	}
	if automationID != "" {
		opts["automation_id"] = automationID
	}
	data, _ := json.Marshal(opts)
	var success string
	if err := runUIAScript(uiaInvokeScript, string(data), &success); err != nil {
		return false, fmt.Errorf("uia_invoke: %w", err)
	}
	return success == "true", nil
}
