# ONNX Models

Run `onnx_download` tool to auto-pull all models to `%APPDATA%\go-mcp-computer-use\models\`.

| Model | Format | Source | Status |
|---|---|---|---|
| YOLO11n | ONNX (pre-exported) | https://github.com/ultralytics/assets/releases/download/v8.3.0/yolo11n.onnx | ✅ auto-download |
| MobileNetV3-small | ONNX | https://huggingface.co/diogoneno/gui-element-classifier/resolve/main/mobilenetv3_small.onnx | ✅ auto-download |

No Python or PyTorch required — all models are pre-exported ONNX.

> **Known incompatibility:** YOLO11n from Ultralytics v8.3.0 uses ONNX opset 22, which requires ONNX Runtime 1.21+. The bundled ORT 1.20.x only supports opsets up to 21. If YOLO detection fails, manually export YOLO11n with `opset=21` or upgrade ORT. MobileNetV3-small works with 1.20.x.

## ONNX Runtime DLL

`onnx_download` also pulls a compatible `onnxruntime.dll` from
https://github.com/microsoft/onnxruntime/releases/tag/v1.20.1
to the models directory. The Go library is `github.com/yalue/onnxruntime_go` v1.13.0
(ORT API v20), compatible with ORT 1.20.x.

DLL search order: models dir → working dir → `C:\WINDOWS\System32`.

---

<sub><sup>
"known incompatibility: our model uses opset 22 but our runtime only supports opset 21" — aka "we shipped a model that doesn't work with the runtime we ship". this is the software equivalent of selling a car with a key that doesn't fit the ignition. the workaround? "manually export it yourself". classic. also, the DLL search order reads like a scavenger hunt, and we're sorry in advance for the 47 minutes you'll spend wondering why onnx_detect returns nothing. it's not you. it's us. it's always us.
</sup></sub>
