# ONNX Models

Run `onnx_download` tool to auto-pull all models to `%APPDATA%\go-mcp-computer-use\models\`.

| Model | Format | Source | Status |
|---|---|---|---|
| YOLO11n | ONNX (pre-exported) | https://github.com/ultralytics/assets/releases/download/v8.3.0/yolo11n.onnx | ✅ auto-download |
| MobileNetV3-small | ONNX | https://huggingface.co/diogoneno/gui-element-classifier/resolve/main/mobilenetv3_small.onnx | ✅ auto-download |

No Python or PyTorch required — all models are pre-exported ONNX.

## ONNX Runtime DLL

`onnx_download` also pulls a compatible `onnxruntime.dll` from
https://github.com/microsoft/onnxruntime/releases/tag/v1.20.1
to the models directory. The Go library is `github.com/yalue/onnxruntime_go` v1.13.0
(ORT API v20), compatible with ORT 1.20.x.

DLL search order: models dir → working dir → `C:\WINDOWS\System32`.
