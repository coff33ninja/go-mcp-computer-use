#Requires -Version 5
param()

$ErrorActionPreference = "Stop"

# ── Step 1: Load WinRT types ──
Write-Host "Loading WinRT types..." -ForegroundColor Cyan

Add-Type -AssemblyName System.Runtime.WindowsRuntime
Add-Type -AssemblyName System.Runtime.InteropServices.WindowsRuntime

$null = [Windows.Storage.StorageFile, Windows.Storage, ContentType=WindowsRuntime]
$null = [Windows.Storage.Streams.RandomAccessStream, Windows.Storage.Streams, ContentType=WindowsRuntime]
$null = [Windows.Graphics.Imaging.BitmapDecoder, Windows.Foundation, ContentType=WindowsRuntime]
$null = [Windows.Graphics.Imaging.SoftwareBitmap, Windows.Foundation, ContentType=WindowsRuntime]
$null = [Windows.Media.Ocr.OcrEngine, Windows.Foundation, ContentType=WindowsRuntime]
$null = [Windows.Media.Ocr.OcrResult, Windows.Foundation, ContentType=WindowsRuntime]
$null = [Windows.Media.Ocr.OcrLine, Windows.Foundation, ContentType=WindowsRuntime]
$null = [Windows.Media.Ocr.OcrWord, Windows.Foundation, ContentType=WindowsRuntime]
$null = [Windows.Globalization.Language, Windows.Foundation, ContentType=WindowsRuntime]

function Get-TypeGuid {
    param([Type]$Type)
    if (-not $Type) { return "NOT FOUND" }
    try { return [System.Runtime.InteropServices.Marshal]::GenerateGuidForType($Type).ToString("D").ToUpperInvariant() }
    catch { return "ERR: $($_.Exception.Message)" }
}

$results = [ordered]@{}

# ── Step 2: Default Interface IIDs ──
Write-Host "`n── Default Interface IIDs ──" -ForegroundColor Yellow

$classInterfaceMap = @{
    "IStorageFile"                       = [Windows.Storage.StorageFile]
    "IBitmapDecoder"                     = [Windows.Graphics.Imaging.BitmapDecoder]
    "ISoftwareBitmap"                    = [Windows.Graphics.Imaging.SoftwareBitmap]
    "IOcrEngine"                         = [Windows.Media.Ocr.OcrEngine]
    "IOcrResult"                         = [Windows.Media.Ocr.OcrResult]
    "IOcrLine"                           = [Windows.Media.Ocr.OcrLine]
    "IOcrWord"                           = [Windows.Media.Ocr.OcrWord]
    "ILanguage"                          = [Windows.Globalization.Language]
}

foreach ($ifaceName in $classInterfaceMap.Keys) {
    $classType = $classInterfaceMap[$ifaceName]
    $iface = $classType.GetInterfaces() | Where-Object { $_.Name -eq $ifaceName } | Select-Object -First 1
    if (-not $iface) { 
        # Try case-insensitive
        $iface = $classType.GetInterfaces() | Where-Object { $_.Name -like "*$ifaceName*" } | Select-Object -First 1
    }
    $results[$ifaceName] = Get-TypeGuid $iface
    Write-Host ("  {0,-55} {1}" -f $ifaceName, $results[$ifaceName])
}

# IRandomAccessStreamWithContentType - try as projected interface directly
try {
    $raType = [Windows.Storage.Streams.IRandomAccessStreamWithContentType]
    $results["IRandomAccessStreamWithContentType"] = Get-TypeGuid $raType
} catch {
    $results["IRandomAccessStreamWithContentType"] = "NOT FOUND"
}
Write-Host ("  {0,-55} {1}" -f "IRandomAccessStreamWithContentType", $results["IRandomAccessStreamWithContentType"])

# ── Step 3: Activation Factory IIDs ──
Write-Host "`n── Activation Factory IIDs ──" -ForegroundColor Yellow

# Statics interfaces ARE separate WinRT types. Search for them by name.
$staticsToFind = @("IStorageFileStatics", "IBitmapDecoderStatics", "IBitmapDecoderStatics2", 
                   "IOcrEngineStatics", "ILanguageFactory", "IRandomAccessStreamStatics")

# Search all loaded assemblies for these interface types
$allAsms = [AppDomain]::CurrentDomain.GetAssemblies()

foreach ($sname in $staticsToFind) {
    $found = $false
    foreach ($asm in $allAsms) {
        try {
            $t = $asm.GetType("Windows.Storage.$sname", $false)
            if (-not $t) { $t = $asm.GetType("Windows.Graphics.Imaging.$sname", $false) }
            if (-not $t) { $t = $asm.GetType("Windows.Media.Ocr.$sname", $false) }
            if (-not $t) { $t = $asm.GetType("Windows.Globalization.$sname", $false) }
            if (-not $t) { $t = $asm.GetType("Windows.Storage.Streams.$sname", $false) }
            
            if ($t) {
                $g = Get-TypeGuid $t
                $results[$sname] = $g
                Write-Host ("  {0,-55} {1}" -f $sname, $g)
                $found = $true
                break
            }
        } catch {}
    }
    if (-not $found) {
        # Broader search: look for the type name anywhere
        foreach ($asm in $allAsms) {
            try {
                foreach ($t in $asm.GetTypes()) {
                    if ($t.Name -eq $sname) {
                        $g = Get-TypeGuid $t
                        $results[$sname] = $g
                        Write-Host ("  {0,-55} {1}  (found in $($t.Namespace) / $($asm.GetName().Name))" -f $sname, $g)
                        $found = $true
                        break
                    }
                }
            } catch {}
            if ($found) { break }
        }
        if (-not $found) {
            $results[$sname] = "NOT FOUND"
            Write-Host ("  {0,-55} NOT FOUND" -f $sname)
        }
    }
}

# ── Step 4: Parameterized Interface IIDs ──
Write-Host "`n── Parameterized Interface IIDs ──" -ForegroundColor Yellow

# Find IAsyncOperation`1 by name across all assemblies
$openAsyncOp = $null
$openVv = $null

foreach ($asm in [AppDomain]::CurrentDomain.GetAssemblies()) {
    try {
        if (-not $openAsyncOp) { $openAsyncOp = $asm.GetType("Windows.Foundation.IAsyncOperation`1") }
        if (-not $openAsyncOp) { 
            # Try alternate names
            foreach ($t in $asm.GetTypes()) {
                if ($t.Name -eq "IAsyncOperation``1") { $openAsyncOp = $t; break }
            }
        }
        if (-not $openVv) { $openVv = $asm.GetType("Windows.Foundation.Collections.IVectorView`1") }
        if (-not $openVv) {
            foreach ($t in $asm.GetTypes()) {
                if ($t.Name -eq "IVectorView``1") { $openVv = $t; break }
            }
        }
    } catch {}
    if ($openAsyncOp -and $openVv) { break }
}

if (-not $openAsyncOp) {
    Write-Host "  IAsyncOperation`1 NOT FOUND - searching every type in every assembly..." -ForegroundColor Yellow
    foreach ($asm in $allAsms) {
        try {
            $types = $asm.GetTypes()
            $matches = $types | Where-Object { $_.Name -like "*AsyncOperation*" -or $_.Name -like "*IAsyncOp*" }
            foreach ($mt in $matches) {
                Write-Host ("    Found: {0}  in {1}" -f $mt.FullName, $asm.GetName().Name) -ForegroundColor DarkGray
                if ($mt.Name -eq "IAsyncOperation``1") { $openAsyncOp = $mt }
            }
        } catch {}
    }
}

if ($openAsyncOp) {
    Write-Host "  Using IAsyncOperation`1 from $($openAsyncOp.Assembly.GetName().Name)" -ForegroundColor Gray
    
    $asyncParamTypes = @(
        @("IAsyncOperation<IStorageFile>", [Windows.Storage.StorageFile]),
        @("IAsyncOperation<IBitmapDecoder>", [Windows.Graphics.Imaging.BitmapDecoder]),
        @("IAsyncOperation<ISoftwareBitmap>", [Windows.Graphics.Imaging.SoftwareBitmap]),
        @("IAsyncOperation<IOcrResult>", [Windows.Media.Ocr.OcrResult]),
        @("IAsyncOperation<IRandomAccessStreamWithContentType>", [Windows.Storage.Streams.IRandomAccessStreamWithContentType])
    )
    
    foreach ($entry in $asyncParamTypes) {
        $name = $entry[0]
        $typeArg = $entry[1]
        try {
            $constructed = $openAsyncOp.MakeGenericType($typeArg)
            $g = Get-TypeGuid $constructed
            $results[$name] = $g
            Write-Host ("  {0,-55} {1}" -f $name, $g)
        } catch {
            $results[$name] = "ERR: $($_.Exception.Message)"
            Write-Host ("  {0,-55} {1}" -f $name, $results[$name])
        }
    }
} else {
    Write-Host "  IAsyncOperation`1 - NOT FOUND" -ForegroundColor Red
}

if ($openVv) {
    Write-Host "  Using IVectorView`1 from $($openVv.Assembly.GetName().Name)" -ForegroundColor Gray
    
    $vvParamTypes = @(
        @("IVectorView<IOcrLine>", [Windows.Media.Ocr.OcrLine]),
        @("IVectorView<IOcrWord>", [Windows.Media.Ocr.OcrWord])
    )
    
    foreach ($entry in $vvParamTypes) {
        $name = $entry[0]
        $typeArg = $entry[1]
        try {
            $constructed = $openVv.MakeGenericType($typeArg)
            $g = Get-TypeGuid $constructed
            $results[$name] = $g
            Write-Host ("  {0,-55} {1}" -f $name, $g)
        } catch {
            $results[$name] = "ERR: $($_.Exception.Message)"
            Write-Host ("  {0,-55} {1}" -f $name, $results[$name])
        }
    }
} else {
    Write-Host "  IVectorView`1 - NOT FOUND" -ForegroundColor Red
}

# ── SUMMARY ──
Write-Host "`n`n╔══════════════════════════════════════════════════════════╗" -ForegroundColor Cyan
Write-Host "║           WinRT COM IIDs for OCR - FINAL               ║" -ForegroundColor Cyan
Write-Host "╚══════════════════════════════════════════════════════════╝" -ForegroundColor Cyan

$cats = @{
    "Activation Factory IIDs" = @("IStorageFileStatics", "IBitmapDecoderStatics", "IBitmapDecoderStatics2", "IOcrEngineStatics", "ILanguageFactory")
    "Default Interface IIDs" = @("IStorageFile", "IRandomAccessStreamWithContentType", "IBitmapDecoder", "ISoftwareBitmap", "IOcrEngine", "IOcrResult", "IOcrLine", "IOcrWord", "ILanguage")
    "Parameterized Interface IIDs" = @("IAsyncOperation<IStorageFile>", "IAsyncOperation<IBitmapDecoder>", "IAsyncOperation<ISoftwareBitmap>", "IAsyncOperation<IOcrResult>", "IAsyncOperation<IRandomAccessStreamWithContentType>", "IVectorView<IOcrLine>", "IVectorView<IOcrWord>")
}

foreach ($catName in $cats.Keys) {
    Write-Host "`n── $catName ──" -ForegroundColor Yellow
    foreach ($key in $cats[$catName]) {
        if ($results.Contains($key)) {
            Write-Host ("  {0,-55} {1}" -f $key, $results[$key])
        } else {
            Write-Host ("  {0,-55} NOT FOUND" -f $key)
        }
    }
}

Write-Host "`nDone." -ForegroundColor Green
