#Requires -Version 5
param(
    [switch]$UpdateDocs
)

$ErrorActionPreference = "Stop"

# ── Step 1: Load WinRT types ──
Write-Host "Loading WinRT types..." -ForegroundColor Cyan

Add-Type -AssemblyName System.Runtime.WindowsRuntime
Add-Type -AssemblyName System.Runtime.InteropServices.WindowsRuntime

function Load-WinRTClass {
    param([string]$TypeName, [string]$Namespace)
    try {
        $null = $TypeName -as [Type]
        return $true
    } catch {
        try {
            $null = [Windows.Storage.StorageFile, Windows.Storage, ContentType=WindowsRuntime]
            # If this worked, the framework is loaded
        } catch {}
    }
    try {
        $null = [System.Type]::GetType("$TypeName, $Namespace, ContentType=WindowsRuntime", $true)
        return $true
    } catch {
        Write-Host ("  WARNING: Could not load {0}" -f $TypeName) -ForegroundColor DarkGray
        return $false
    }
}

# OCR pipeline
$null = [Windows.Storage.StorageFile, Windows.Storage, ContentType=WindowsRuntime]
$null = [Windows.Storage.Streams.RandomAccessStream, Windows.Storage.Streams, ContentType=WindowsRuntime]
$null = [Windows.Storage.Streams.InMemoryRandomAccessStream, Windows.Storage.Streams, ContentType=WindowsRuntime]
$null = [Windows.Storage.Streams.DataWriter, Windows.Storage.Streams, ContentType=WindowsRuntime]
$null = [Windows.Storage.Streams.DataReader, Windows.Storage.Streams, ContentType=WindowsRuntime]
$null = [Windows.Storage.Pickers.FileOpenPicker, Windows.Storage.Pickers, ContentType=WindowsRuntime]
$null = [Windows.Storage.Pickers.FileSavePicker, Windows.Storage.Pickers, ContentType=WindowsRuntime]
$null = [Windows.Graphics.Imaging.BitmapDecoder, Windows.Foundation, ContentType=WindowsRuntime]
$null = [Windows.Graphics.Imaging.BitmapEncoder, Windows.Foundation, ContentType=WindowsRuntime]
$null = [Windows.Graphics.Imaging.SoftwareBitmap, Windows.Foundation, ContentType=WindowsRuntime]
$null = [Windows.Media.Ocr.OcrEngine, Windows.Foundation, ContentType=WindowsRuntime]
$null = [Windows.Media.Ocr.OcrResult, Windows.Foundation, ContentType=WindowsRuntime]
$null = [Windows.Media.Ocr.OcrLine, Windows.Foundation, ContentType=WindowsRuntime]
$null = [Windows.Media.Ocr.OcrWord, Windows.Foundation, ContentType=WindowsRuntime]
$null = [Windows.Globalization.Language, Windows.Foundation, ContentType=WindowsRuntime]
# Audio / Devices
$null = [Windows.Media.Devices.MediaDevice, Windows.Foundation, ContentType=WindowsRuntime]
$null = [Windows.Devices.Enumeration.DeviceInformation, Windows.Foundation, ContentType=WindowsRuntime]
$null = [Windows.Devices.Enumeration.DeviceInformationCollection, Windows.Foundation, ContentType=WindowsRuntime]
# Power / System
$null = [Windows.System.Power.PowerManager, Windows.Foundation, ContentType=WindowsRuntime]
$null = [Windows.System.Launcher, Windows.Foundation, ContentType=WindowsRuntime]
$null = [Windows.System.UserProfile.UserProfilePersonalizationSettings, Windows.Foundation, ContentType=WindowsRuntime]
$null = [Windows.System.UserProfile.UserInformation, Windows.Foundation, ContentType=WindowsRuntime]
$null = [Windows.System.Diagnostics.ProcessDiagnosticInfo, Windows.Foundation, ContentType=WindowsRuntime]
# Display
$null = [Windows.Graphics.Display.DisplayInformation, Windows.Foundation, ContentType=WindowsRuntime]
# Notifications
$null = [Windows.UI.Notifications.ToastNotificationManager, Windows.Foundation, ContentType=WindowsRuntime]
$null = [Windows.UI.Notifications.ToastNotification, Windows.Foundation, ContentType=WindowsRuntime]
# Clipboard
$null = [Windows.ApplicationModel.DataTransfer.Clipboard, Windows.Foundation, ContentType=WindowsRuntime]
$null = [Windows.ApplicationModel.DataTransfer.DataPackage, Windows.Foundation, ContentType=WindowsRuntime]
# Media control
$null = [Windows.Media.Control.GlobalSystemMediaTransportControlsSessionManager, Windows.Foundation, ContentType=WindowsRuntime]

function Get-TypeGuid {
    param([Type]$Type)
    if (-not $Type) { return "NOT FOUND" }
    try { return [System.Runtime.InteropServices.Marshal]::GenerateGuidForType($Type).ToString("D").ToUpperInvariant() }
    catch { return "ERR: $($_.Exception.Message)" }
}

$results = [ordered]@{}
$allAsms = [AppDomain]::CurrentDomain.GetAssemblies()

# ── Helper: find type by name across all assemblies ──
function Find-TypeByName {
    param([string]$Name)
    foreach ($asm in $allAsms) {
        try {
            foreach ($t in $asm.GetTypes()) {
                if ($t.Name -eq $Name -or $t.FullName -eq $Name) { return $t }
            }
        } catch {}
    }
    return $null
}

# ── Helper: probe class interfaces ──
function Probe-ClassInterfaces {
    param(
        [string]$Label,
        [Type]$ClassType,
        [string[]]$KnownInterfaceNames
    )
    Write-Host "`n── $Label ──" -ForegroundColor Yellow
    if (-not $ClassType) {
        Write-Host "  (class not loaded)"
        return
    }
    $ifaces = $ClassType.GetInterfaces()
    if ($KnownInterfaceNames) {
        # Only probe specific named interfaces
        foreach ($iname in $KnownInterfaceNames) {
            $match = $ifaces | Where-Object { $_.Name -eq $iname -or $_.Name -like "*$iname*" } | Select-Object -First 1
            if (-not $match) {
                # Broader search
                $match = $ifaces | Where-Object { $_.Name -match $iname } | Select-Object -First 1
            }
            if ($match) {
                $guid = Get-TypeGuid $match
                $results["${Label}:$iname"] = $guid
                Write-Host ("  {0,-65} {1}" -f "${Label}.$iname", $guid)
            } else {
                Write-Host ("  {0,-65} NOT FOUND" -f "${Label}.$iname")
            }
        }
    } else {
        # Show ALL interfaces
        foreach ($iface in $ifaces) {
            $guid = Get-TypeGuid $iface
            $results["${Label}:$($iface.Name)"] = $guid
            Write-Host ("  {0,-65} {1}" -f "${Label}.$($iface.Name)", $guid)
        }
    }
}

# ── Step 2: OCR Pipeline Interfaces ──
Probe-ClassInterfaces -Label "OCR" -ClassType ([Windows.Media.Ocr.OcrEngine]) -KnownInterfaceNames @("IOcrEngine")
Probe-ClassInterfaces -Label "OCRResult" -ClassType ([Windows.Media.Ocr.OcrResult]) -KnownInterfaceNames @("IOcrResult")
Probe-ClassInterfaces -Label "OCRLine" -ClassType ([Windows.Media.Ocr.OcrLine]) -KnownInterfaceNames @("IOcrLine")
Probe-ClassInterfaces -Label "OCRWord" -ClassType ([Windows.Media.Ocr.OcrWord]) -KnownInterfaceNames @("IOcrWord")

# ── Step 3: Storage & Stream Interfaces ──
Probe-ClassInterfaces -Label "StorageFile" -ClassType ([Windows.Storage.StorageFile]) -KnownInterfaceNames @("IStorageFile")
Probe-ClassInterfaces -Label "RandomAccessStream" -ClassType ([Windows.Storage.Streams.RandomAccessStream]) -KnownInterfaceNames @("IRandomAccessStreamWithContentType")
Probe-ClassInterfaces -Label "InMemoryRandomAccessStream" -ClassType ([Windows.Storage.Streams.InMemoryRandomAccessStream]) -KnownInterfaceNames @()
Probe-ClassInterfaces -Label "DataWriter" -ClassType ([Windows.Storage.Streams.DataWriter]) -KnownInterfaceNames @("IDataWriter")
Probe-ClassInterfaces -Label "DataReader" -ClassType ([Windows.Storage.Streams.DataReader]) -KnownInterfaceNames @("IDataReader")
Probe-ClassInterfaces -Label "FileOpenPicker" -ClassType ([Windows.Storage.Pickers.FileOpenPicker]) -KnownInterfaceNames @("IFileOpenPicker")
Probe-ClassInterfaces -Label "FileSavePicker" -ClassType ([Windows.Storage.Pickers.FileSavePicker]) -KnownInterfaceNames @("IFileSavePicker")

# ── Step 4: Bitmap Interfaces ──
Probe-ClassInterfaces -Label "BitmapDecoder" -ClassType ([Windows.Graphics.Imaging.BitmapDecoder]) -KnownInterfaceNames @("IBitmapDecoder", "IBitmapFrame", "IBitmapFrameWithSoftwareBitmap")
Probe-ClassInterfaces -Label "BitmapEncoder" -ClassType ([Windows.Graphics.Imaging.BitmapEncoder]) -KnownInterfaceNames @("IBitmapEncoder")
Probe-ClassInterfaces -Label "SoftwareBitmap" -ClassType ([Windows.Graphics.Imaging.SoftwareBitmap]) -KnownInterfaceNames @("ISoftwareBitmap")

# ── Step 5: Globalization Interfaces ──
Probe-ClassInterfaces -Label "Language" -ClassType ([Windows.Globalization.Language]) -KnownInterfaceNames @("ILanguage")

# ── Step 6: Audio / Device Interfaces ──
Probe-ClassInterfaces -Label "MediaDevice" -ClassType ([Windows.Media.Devices.MediaDevice]) -KnownInterfaceNames @("IMediaDevice")
Probe-ClassInterfaces -Label "DeviceInformation" -ClassType ([Windows.Devices.Enumeration.DeviceInformation]) -KnownInterfaceNames @("IDeviceInformation")
Probe-ClassInterfaces -Label "DeviceInfoCollection" -ClassType ([Windows.Devices.Enumeration.DeviceInformationCollection]) -KnownInterfaceNames @("IDeviceInformationCollection")

# ── Step 7: Power / System Interfaces ──
Probe-ClassInterfaces -Label "PowerManager" -ClassType ([Windows.System.Power.PowerManager]) -KnownInterfaceNames @("IPowerManager")
Probe-ClassInterfaces -Label "Launcher" -ClassType ([Windows.System.Launcher]) -KnownInterfaceNames @("ILauncher")
Probe-ClassInterfaces -Label "UserProfileSettings" -ClassType ([Windows.System.UserProfile.UserProfilePersonalizationSettings]) -KnownInterfaceNames @("IUserProfilePersonalizationSettings")
Probe-ClassInterfaces -Label "UserInformation" -ClassType ([Windows.System.UserProfile.UserInformation]) -KnownInterfaceNames @("IUserInformation")
Probe-ClassInterfaces -Label "ProcessDiagnosticInfo" -ClassType ([Windows.System.Diagnostics.ProcessDiagnosticInfo]) -KnownInterfaceNames @("IProcessDiagnosticInfo")

# ── Step 8: Display Interfaces ──
Probe-ClassInterfaces -Label "DisplayInformation" -ClassType ([Windows.Graphics.Display.DisplayInformation]) -KnownInterfaceNames @("IDisplayInformation")

# ── Step 9: Notification Interfaces ──
Probe-ClassInterfaces -Label "ToastNotificationManager" -ClassType ([Windows.UI.Notifications.ToastNotificationManager]) -KnownInterfaceNames @("IToastNotificationManager")
Probe-ClassInterfaces -Label "ToastNotification" -ClassType ([Windows.UI.Notifications.ToastNotification]) -KnownInterfaceNames @("IToastNotification")

# ── Step 10: Clipboard Interfaces ──
Probe-ClassInterfaces -Label "Clipboard" -ClassType ([Windows.ApplicationModel.DataTransfer.Clipboard]) -KnownInterfaceNames @("IClipboard")
Probe-ClassInterfaces -Label "DataPackage" -ClassType ([Windows.ApplicationModel.DataTransfer.DataPackage]) -KnownInterfaceNames @("IDataPackage")

# ── Step 11: Media Control Interfaces ──
Probe-ClassInterfaces -Label "GSMTCSessionManager" -ClassType ([Windows.Media.Control.GlobalSystemMediaTransportControlsSessionManager]) -KnownInterfaceNames @("IGlobalSystemMediaTransportControlsSessionManager", "IGlobalSystemMediaTransportControlsSession")

# ── Step 12: Standalone Interface IIDs (direct type load) ──
Write-Host "`n── Standalone Interface IIDs (direct type load) ──" -ForegroundColor Yellow

$standaloneInterfaces = @(
    @("IRandomAccessStreamWithContentType", [Windows.Storage.Streams.IRandomAccessStreamWithContentType]),
    @("IRandomAccessStream", [Windows.Storage.Streams.IRandomAccessStream]),
    @("IInputStream", [Windows.Storage.Streams.IInputStream]),
    @("IOutputStream", [Windows.Storage.Streams.IOutputStream]),
    @("IDisposable", [System.IDisposable])
)

foreach ($entry in $standaloneInterfaces) {
    $iname = $entry[0]; $itype = $entry[1]
    try {
        $g = Get-TypeGuid $itype
        $results[$iname] = $g
        Write-Host ("  {0,-65} {1}" -f $iname, $g)
    } catch {
        $results[$iname] = "NOT FOUND"
        Write-Host ("  {0,-65} NOT FOUND" -f $iname)
    }
}

# ── Step 13: Activation Factory IIDs ──
Write-Host "`n── Activation Factory / Statics IIDs ──" -ForegroundColor Yellow

$factoryNames = @(
    "IStorageFileStatics",
    "IStorageFileStatics2",
    "IBitmapDecoderStatics",
    "IBitmapDecoderStatics2",
    "IBitmapEncoderStatics",
    "IOcrEngineStatics",
    "ILanguageFactory",
    "ILanguageStatics",
    "IRandomAccessStreamStatics",
    "IDeviceInformationStatics",
    "IDeviceInformationCollectionFactory",
    "IMediaDeviceStatics",
    "IPowerManagerStatics",
    "ILauncherStatics",
    "IUserProfilePersonalizationSettingsStatics",
    "IUserInformationStatics",
    "IDisplayInformationStatics",
    "IToastNotificationManagerStatics",
    "IClipboardStatics",
    "IFileOpenPickerStatics",
    "IFileSavePickerStatics",
    "IGlobalSystemMediaTransportControlsSessionManagerStatics",
    "IProcessDiagnosticInfoStatics"
)

$knownNamespaces = @(
    "Windows.Storage",
    "Windows.Storage.Streams",
    "Windows.Storage.Pickers",
    "Windows.Graphics.Imaging",
    "Windows.Media.Ocr",
    "Windows.Globalization",
    "Windows.Media.Devices",
    "Windows.Devices.Enumeration",
    "Windows.System.Power",
    "Windows.System",
    "Windows.System.UserProfile",
    "Windows.System.Diagnostics",
    "Windows.Graphics.Display",
    "Windows.UI.Notifications",
    "Windows.ApplicationModel.DataTransfer",
    "Windows.Media.Control"
)

foreach ($sname in $factoryNames) {
    $found = $false
    # Search in known namespaces first
    foreach ($ns in $knownNamespaces) {
        try {
            $fqn = "$ns.$sname"
            $t = [System.Type]::GetType($fqn, $false)
            if ($t) {
                $g = Get-TypeGuid $t
                $results[$sname] = $g
                Write-Host ("  {0,-65} {1}" -f "$sname ($ns)", $g)
                $found = $true
                break
            }
        } catch {}
    }
    if (-not $found) {
        # Broad search across all assemblies
        $t = Find-TypeByName -Name $sname
        if ($t) {
            $g = Get-TypeGuid $t
            $results[$sname] = $g
            Write-Host ("  {0,-65} {1}  (found in {2})" -f $sname, $g, $t.Namespace)
        } else {
            $results[$sname] = "NOT FOUND"
            Write-Host ("  {0,-65} NOT FOUND" -f $sname)
        }
    }
}

# ── Step 13: Parameterized Interface IIDs ──
Write-Host "`n── Parameterized Interface IIDs ──" -ForegroundColor Yellow

# Search all assemblies for generic WinRT types by scanning every type
$openAsyncOp = $null
$openVv = $null
$openVect = $null
$openIter = $null

foreach ($asm in $allAsms) {
    try {
        if (-not $openAsyncOp) { $openAsyncOp = $asm.GetType("Windows.Foundation.IAsyncOperation`1") }
        if (-not $openVv) { $openVv = $asm.GetType("Windows.Foundation.Collections.IVectorView`1") }
        if (-not $openVect) { $openVect = $asm.GetType("Windows.Foundation.Collections.IVector`1") }
        if (-not $openIter) { $openIter = $asm.GetType("Windows.Foundation.Collections.IIterable`1") }
    } catch {}
    if ($openAsyncOp -and $openVv) { break }
}

# Fallback: brute-force scan if GetType didn't work
if (-not $openAsyncOp -or -not $openVv) {
    Write-Host "  GetType lookup failed, scanning all types..." -ForegroundColor Gray
    foreach ($asm in $allAsms) {
        try {
            foreach ($t in $asm.GetTypes()) {
                if (-not $openAsyncOp -and $t.Name -eq "IAsyncOperation``1") { $openAsyncOp = $t }
                if (-not $openVv -and $t.Name -eq "IVectorView``1") { $openVv = $t }
                if (-not $openVect -and $t.Name -eq "IVector``1") { $openVect = $t }
                if (-not $openIter -and $t.Name -eq "IIterable``1") { $openIter = $t }
                if ($openAsyncOp -and $openVv) { break }
            }
        } catch {}
        if ($openAsyncOp -and $openVv) { break }
    }
}

if ($openAsyncOp) {
    Write-Host "  Using IAsyncOperation`1 from $($openAsyncOp.Assembly.GetName().Name)" -ForegroundColor Gray
    $asyncTypes = @(
        @("IAsyncOperation<IStorageFile>", [Windows.Storage.StorageFile]),
        @("IAsyncOperation<IBitmapDecoder>", [Windows.Graphics.Imaging.BitmapDecoder]),
        @("IAsyncOperation<ISoftwareBitmap>", [Windows.Graphics.Imaging.SoftwareBitmap]),
        @("IAsyncOperation<IOcrResult>", [Windows.Media.Ocr.OcrResult]),
        @("IAsyncOperation<IRandomAccessStreamWithContentType>", [Windows.Storage.Streams.IRandomAccessStreamWithContentType]),
        @("IAsyncOperation<DeviceInformationCollection>", [Windows.Devices.Enumeration.DeviceInformationCollection]),
        @("IAsyncOperation<DataPackage>", [Windows.ApplicationModel.DataTransfer.DataPackage])
    )
    foreach ($entry in $asyncTypes) {
        $name = $entry[0]; $typeArg = $entry[1]
        try {
            $constructed = $openAsyncOp.MakeGenericType($typeArg)
            $g = Get-TypeGuid $constructed
            $results["Async:$name"] = $g
            Write-Host ("  {0,-65} {1}" -f $name, $g)
        } catch {
            Write-Host ("  {0,-65} ERR: {1}" -f $name, $_.Exception.Message)
        }
    }
} else {
    Write-Host "  IAsyncOperation`1 - NOT FOUND" -ForegroundColor Red
}

if ($openVv) {
    Write-Host "  Using IVectorView`1 from $($openVv.Assembly.GetName().Name)" -ForegroundColor Gray
    $vvTypes = @(
        @("IVectorView<IOcrLine>", [Windows.Media.Ocr.OcrLine]),
        @("IVectorView<IOcrWord>", [Windows.Media.Ocr.OcrWord]),
        @("IVectorView<DeviceInformation>", [Windows.Devices.Enumeration.DeviceInformation])
    )
    foreach ($entry in $vvTypes) {
        $name = $entry[0]; $typeArg = $entry[1]
        try {
            $constructed = $openVv.MakeGenericType($typeArg)
            $g = Get-TypeGuid $constructed
            $results["VV:$name"] = $g
            Write-Host ("  {0,-65} {1}" -f $name, $g)
        } catch {
            Write-Host ("  {0,-65} ERR: {1}" -f $name, $_.Exception.Message)
        }
    }
} else {
    Write-Host "  IVectorView`1 - NOT FOUND" -ForegroundColor Red
}

if ($openVect) {
    Write-Host "  Using IVector`1 from $($openVect.Assembly.GetName().Name)" -ForegroundColor Gray
    $vectTypes = @(
        @("IVector<OcrWord>", [Windows.Media.Ocr.OcrWord])
    )
    foreach ($entry in $vectTypes) {
        $name = $entry[0]; $typeArg = $entry[1]
        try {
            $constructed = $openVect.MakeGenericType($typeArg)
            $g = Get-TypeGuid $constructed
            $results["Vect:$name"] = $g
            Write-Host ("  {0,-65} {1}" -f $name, $g)
        } catch {
            Write-Host ("  {0,-65} ERR: {1}" -f $name, $_.Exception.Message)
        }
    }
} else {
    Write-Host "  IVector`1 - NOT FOUND" -ForegroundColor Red
}

# ── SUMMARY ──
Write-Host "`n`n╔══════════════════════════════════════════════════════════════════════╗" -ForegroundColor Cyan
Write-Host "║           WinRT COM IIDs — Full Discovery                        ║" -ForegroundColor Cyan
Write-Host "╚══════════════════════════════════════════════════════════════════════╝" -ForegroundColor Cyan

# Group results by area for summary
# Keys can appear as "InterfaceName" (statics/factories) or "Prefix.InterfaceName" (class interfaces)
$summaryCats = @{
    "Storage & Streams"   = @("IStorageFile", "IStorageFileStatics", "IStorageFileStatics2", "IRandomAccessStreamWithContentType", "IRandomAccessStream", "IRandomAccessStreamStatics", "IInputStream", "IOutputStream", "IDataWriter", "IDataReader", "IFileOpenPicker", "IFileSavePicker", "IFileOpenPickerStatics", "IFileSavePickerStatics")
    "Bitmap / Imaging"    = @("IBitmapDecoder", "IBitmapDecoderStatics", "IBitmapDecoderStatics2", "IBitmapFrame", "IBitmapFrameWithSoftwareBitmap", "IBitmapEncoder", "IBitmapEncoderStatics", "ISoftwareBitmap")
    "OCR"                 = @("IOcrEngine", "IOcrEngineStatics", "IOcrResult", "IOcrLine", "IOcrWord")
    "Globalization"       = @("ILanguage", "ILanguageFactory", "ILanguageStatics")
    "Devices & Audio"     = @("IDeviceInformation", "IDeviceInformationStatics", "IMediaDeviceStatics")
    "Power / System"      = @("IPowerManagerStatics", "ILauncherStatics", "IUserProfilePersonalizationSettings", "IUserProfilePersonalizationSettingsStatics", "IUserInformationStatics", "IProcessDiagnosticInfo", "IProcessDiagnosticInfoStatics")
    "Display"             = @("IDisplayInformation", "IDisplayInformationStatics")
    "Notifications"       = @("IToastNotificationManagerStatics", "IToastNotification")
    "Clipboard"           = @("IClipboardStatics", "IDataPackage")
    "Media Control"       = @("IGlobalSystemMediaTransportControlsSessionManager", "IGlobalSystemMediaTransportControlsSessionManagerStatics", "IGlobalSystemMediaTransportControlsSession")
    "Standalone Interfaces" = @("IRandomAccessStreamWithContentType", "IRandomAccessStream", "IInputStream", "IOutputStream", "IDisposable")
    "Parameterized Interfaces" = @()
}

function Get-SummaryValue {
    param([string]$Key)
    if ($results.Contains($Key)) { return $results[$Key] }
    # Search all keys for suffix match (keys use format "Label:IfaceName" or "Async:IfaceName" or flat)
    foreach ($rk in $results.Keys) {
        $parts = $rk -split ':'
        if ($parts.Count -eq 2 -and $parts[1] -eq $Key) { return $results[$rk] }
        if ($rk -eq $Key) { return $results[$rk] }
    }
    return "NOT FOUND"
}

foreach ($catName in $summaryCats.Keys) {
    $catKeys = $summaryCats[$catName]
    Write-Host "`n── $catName ──" -ForegroundColor Yellow
    foreach ($key in $catKeys) {
        $val = Get-SummaryValue -Key $key
        Write-Host ("  {0,-65} {1}" -f $key, $val)
    }
}

# Additional parameterized IIDs not in categories above
Write-Host "`n── Parameterized Interface IIDs ──" -ForegroundColor Yellow
foreach ($rk in $results.Keys) {
    if ($rk -like 'Async:*' -or $rk -like 'VV:*' -or $rk -like 'Vect:*') {
        Write-Host ("  {0,-65} {1}" -f $rk, $results[$rk])
    }
}

# ── Step 14: Update Markdown Docs ──
if ($UpdateDocs) {
    Write-Host "`n── Updating docs/reference/com-patterns.md ──" -ForegroundColor Cyan

    # Hardcoded IIDs (not discoverable via reflection)
    $hardcoded = @{
        "IID_IInspectable"       = "{AF86E2E0-B12D-4c6a-9C5A-D7AA65101E90}"
        "IID_IActivationFactory" = "{00000035-0000-0000-C000-000000000046}"
        "IID_IAsyncInfo"         = "{00000036-0000-0000-C000-000000000046}"
    }

    # Result key → Go variable name mapping
    $resultKeyMap = @{
        "OCR:IOcrEngine"                                       = "IID_IOcrEngine"
        "OCRResult:IOcrResult"                                 = "IID_IOcrResult"
        "OCRLine:IOcrLine"                                     = "IID_IOcrLine"
        "OCRWord:IOcrWord"                                     = "IID_IOcrWord"
        "StorageFile:IStorageFile"                             = "IID_IStorageFile"
        "RandomAccessStream:IRandomAccessStreamWithContentType" = "IID_IRandomAccessStreamWithContentType"
        "DataWriter:IDataWriter"                               = "IID_IDataWriter"
        "DataReader:IDataReader"                               = "IID_IDataReader"
        "FileOpenPicker:IFileOpenPicker"                       = "IID_IFileOpenPicker"
        "FileSavePicker:IFileSavePicker"                       = "IID_IFileSavePicker"
        "BitmapDecoder:IBitmapDecoder"                         = "IID_IBitmapDecoder"
        "BitmapDecoder:IBitmapFrame"                           = "IID_IBitmapFrame"
        "BitmapDecoder:IBitmapFrameWithSoftwareBitmap"         = "IID_IBitmapFrameWithSoftwareBitmap"
        "BitmapEncoder:IBitmapEncoder"                         = "IID_IBitmapEncoder"
        "SoftwareBitmap:ISoftwareBitmap"                       = "IID_ISoftwareBitmap"
        "Language:ILanguage"                                   = "IID_ILanguage"
        "DeviceInformation:IDeviceInformation"                 = "IID_IDeviceInformation"
        "PowerManager:IPowerManager"                           = "IID_IPowerManagerStatics"
        "Launcher:ILauncher"                                   = "IID_ILauncherStatics"
        "UserProfileSettings:IUserProfilePersonalizationSettings" = "IID_IUserProfilePersonalizationSettings"
        "UserInformation:IUserInformation"                     = "IID_IUserInformationStatics"
        "ProcessDiagnosticInfo:IProcessDiagnosticInfo"         = "IID_IProcessDiagnosticInfo"
        "DisplayInformation:IDisplayInformation"               = "IID_IDisplayInformation"
        "ToastNotification:IToastNotification"                 = "IID_IToastNotification"
        "DataPackage:IDataPackage"                             = "IID_IDataPackage"
        "GSMTCSessionManager:IGlobalSystemMediaTransportControlsSessionManager" = "IID_IGlobalSystemMediaTransportControlsSessionManager"
        "IStorageFileStatics"                                  = "IID_IStorageFileStatics"
        "IStorageFileStatics2"                                 = "IID_IStorageFileStatics2"
        "IBitmapDecoderStatics"                                = "IID_IBitmapDecoderStatics"
        "IBitmapDecoderStatics2"                               = "IID_IBitmapDecoderStatics2"
        "IBitmapEncoderStatics"                                = "IID_IBitmapEncoderStatics"
        "IOcrEngineStatics"                                    = "IID_IOcrEngineStatics"
        "ILanguageFactory"                                     = "IID_ILanguageFactory"
        "ILanguageStatics"                                     = "IID_ILanguageStatics"
        "IRandomAccessStreamStatics"                           = "IID_IRandomAccessStreamStatics"
        "IDeviceInformationStatics"                            = "IID_IDeviceInformationStatics"
        "IMediaDeviceStatics"                                  = "IID_IMediaDeviceStatics"
        "IUserProfilePersonalizationSettingsStatics"           = "IID_IUserProfilePersonalizationSettingsStatics"
        "IProcessDiagnosticInfoStatics"                        = "IID_IProcessDiagnosticInfoStatics"
        "IDisplayInformationStatics"                           = "IID_IDisplayInformationStatics"
        "IToastNotificationManagerStatics"                     = "IID_IToastNotificationManagerStatics"
        "IClipboardStatics"                                    = "IID_IClipboardStatics"
        "IFileOpenPickerStatics"                               = "IID_IFileOpenPickerStatics"
        "IFileSavePickerStatics"                               = "IID_IFileSavePickerStatics"
        "IGlobalSystemMediaTransportControlsSessionManagerStatics" = "IID_IGlobalSystemMediaTransportControlsSessionManagerStatics"
    }

    # Standalone interfaces (probed via direct type, not class interfaces)
    $resultKeyMap["IRandomAccessStreamWithContentType"] = "IID_IRandomAccessStreamWithContentType"  # from standalone step
    $resultKeyMap["IRandomAccessStream"]                = "IID_IRandomAccessStream"
    $resultKeyMap["IInputStream"]                       = "IID_IInputStream"
    $resultKeyMap["IOutputStream"]                      = "IID_IOutputStream"

    # Purpose mapping (Go variable name → short description)
    $purposeMap = [ordered]@{
        "IID_IInspectable"       = "Base WinRT interface"
        "IID_IActivationFactory" = "Activation factory"
        "IID_IAsyncInfo"         = "Async operation status"
        "IID_IStorageFileStatics" = "StorageFile factory"
        "IID_IStorageFileStatics2" = "StorageFile factory v2"
        "IID_IStorageFile"       = "StorageFile instance"
        "IID_IBitmapDecoderStatics" = "BitmapDecoder factory"
        "IID_IBitmapDecoderStatics2" = "BitmapDecoder factory v2"
        "IID_IBitmapDecoder"     = "BitmapDecoder instance"
        "IID_IBitmapFrame"       = "Bitmap frame"
        "IID_IBitmapFrameWithSoftwareBitmap" = "SoftwareBitmap extraction"
        "IID_IBitmapEncoder"     = "BitmapEncoder instance"
        "IID_IBitmapEncoderStatics" = "BitmapEncoder factory"
        "IID_ISoftwareBitmap"    = "SoftwareBitmap instance"
        "IID_IRandomAccessStream" = "RandomAccessStream base"
        "IID_IRandomAccessStreamStatics" = "RandomAccessStream factory"
        "IID_IRandomAccessStreamWithContentType" = "Stream with content type"
        "IID_IInputStream"       = "Input stream"
        "IID_IOutputStream"      = "Output stream"
        "IID_IDataWriter"        = "DataWriter"
        "IID_IDataReader"        = "DataReader"
        "IID_IFileOpenPicker"    = "FileOpenPicker instance"
        "IID_IFileOpenPickerStatics" = "FileOpenPicker factory"
        "IID_IFileSavePicker"    = "FileSavePicker instance"
        "IID_IFileSavePickerStatics" = "FileSavePicker factory"
        "IID_IOcrEngineStatics"  = "OcrEngine factory"
        "IID_IOcrEngine"         = "OcrEngine instance"
        "IID_IOcrResult"         = "OCR result"
        "IID_IOcrLine"           = "OCR line"
        "IID_IOcrWord"           = "OCR word"
        "IID_ILanguageFactory"   = "Language factory"
        "IID_ILanguage"          = "Language instance"
        "IID_ILanguageStatics"   = "Language statics"
        "IID_IDeviceInformation" = "DeviceInformation instance"
        "IID_IDeviceInformationStatics" = "DeviceInformation factory"
        "IID_IMediaDeviceStatics" = "MediaDevice factory"
        "IID_IPowerManagerStatics" = "PowerManager factory"
        "IID_ILauncherStatics"   = "Launcher factory"
        "IID_IUserProfilePersonalizationSettings" = "Personalization settings"
        "IID_IUserProfilePersonalizationSettingsStatics" = "Personalization settings factory"
        "IID_IUserInformationStatics" = "UserInformation factory"
        "IID_IProcessDiagnosticInfo" = "Process info instance"
        "IID_IProcessDiagnosticInfoStatics" = "Process info factory"
        "IID_IDisplayInformation" = "Display info instance"
        "IID_IDisplayInformationStatics" = "Display info factory"
        "IID_IToastNotification" = "Toast notification"
        "IID_IToastNotificationManagerStatics" = "Toast notification factory"
        "IID_IClipboardStatics"  = "Clipboard factory"
        "IID_IDataPackage"       = "DataPackage instance"
        "IID_IGlobalSystemMediaTransportControlsSessionManager" = "Media session manager"
        "IID_IGlobalSystemMediaTransportControlsSessionManagerStatics" = "Media session manager factory"
    }

    # Build IID value lookup
    $iidValue = @{}
    foreach ($rk in $resultKeyMap.Keys) {
        $gv = $resultKeyMap[$rk]
        if ($results.Contains($rk)) {
            $val = $results[$rk]
            if ($val -ne "NOT FOUND" -and -not $val.StartsWith("ERR:")) {
                $iidValue[$gv] = $val
            }
        }
    }
    # Fallback pass: for any Go var still missing, try flat key (strip IID_ prefix)
    foreach ($gv in $purposeMap.Keys) {
        if (-not $iidValue.Contains($gv)) {
            $flatKey = $gv -replace '^IID_', ''
            if ($results.Contains($flatKey)) {
                $val = $results[$flatKey]
                if ($val -ne "NOT FOUND" -and -not $val.StartsWith("ERR:")) {
                    $iidValue[$gv] = $val
                }
            }
        }
    }
    foreach ($hk in $hardcoded.Keys) {
        $iidValue[$hk] = $hardcoded[$hk]
    }

    # Generate markdown table rows
    $rows = @()
    $rows += "| IID | Value | Purpose | Status |"
    $rows += "|-----|-------|---------|--------|"
    foreach ($gv in $purposeMap.Keys) {
        $val = if ($iidValue.Contains($gv)) { $iidValue[$gv] } else { "NOT FOUND" }
        $purpose = $purposeMap[$gv]
        $rows += "| ``{0}`` | {1} | {2} | unused |" -f $gv, "``$val``", $purpose
    }

    $docPath = Join-Path $PSScriptRoot "..\docs\reference\com-patterns.md"
    if (Test-Path -LiteralPath $docPath) {
        $content = Get-Content -LiteralPath $docPath -Raw
        $tableBlock = ($rows -join "`n")
        $newContent = $content -replace '(?s)<!-- IID_TABLE_START -->.*?<!-- IID_TABLE_END -->', "<!-- IID_TABLE_START -->`n$tableBlock`n<!-- IID_TABLE_END -->"
        Set-Content -LiteralPath $docPath -Value $newContent -NoNewline
        Write-Host ("  Updated {0} with {1} IID entries." -f $docPath, $purposeMap.Count) -ForegroundColor Green
        Write-Host "  Updating usage statuses via verify-iid-usage.go -update..." -ForegroundColor Cyan
        go run (Join-Path $PSScriptRoot "verify-iid-usage.go") -update
    } else {
        Write-Host ("  WARNING: {0} not found, cannot update docs." -f $docPath) -ForegroundColor Yellow
    }
}

Write-Host "`nDone. To add more WinRT types, extend the type-loading section at the top of this script." -ForegroundColor Green
