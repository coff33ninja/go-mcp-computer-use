# ONNX Models

Run `onnx_download` tool to auto-pull all models to `%APPDATA%\go-mcp-computer-use\models\`.

| Model | Format | Source | Status |
|---|---|---|---|
| YOLO11n | ONNX (pre-exported) | https://github.com/ultralytics/assets/releases/download/v8.3.0/yolo11n.onnx | ✅ auto-download |
| MobileNetV3-small | ONNX | https://huggingface.co/diogoneno/gui-element-classifier/resolve/main/mobilenetv3_small.onnx | ✅ auto-download |

No Python or PyTorch required — all models are pre-exported ONNX.

> **Known incompatibility:** YOLO11n from Ultralytics v8.3.0 uses ONNX opset 22, which requires ONNX Runtime 1.21+. The bundled ORT 1.20.x only supports opsets up to 21. If YOLO detection fails, manually export YOLO11n with `opset=21` or upgrade ORT. MobileNetV3-small works with 1.20.x.

## Transformer Model (Go-native)

The action prediction transformer is trained entirely in Go via [Gorgonia](https://gorgonia.org). No ONNX, no Python — the model is persisted as a Go gob file.

| Property | Value |
|---|---|
| Format | `model.gob` (Go binary serialization) |
| Location | Same directory as `datalog.db` (usually `%APPDATA%/go-mcp-computer-use/`) |
| Architecture | 64-dim embeddings, 2 layers, 2 attention heads |
| Parameters | ~14K (tiny — trains in seconds, loads in milliseconds) |
| Input | Tokenized OCR text + 7-feature spatial encoding (DPI-aware) |
| Output | Tool probabilities (softmax) + normalized coordinates (sigmoid) |
| Training | Adam optimizer, MSE loss, from `training_pairs` table in SQLite |
| Auto-load | On server start: loads `model.gob` if present, otherwise trains from datalog |
| Self-improving | Each session's training pairs feed back into the model automatically |

### How It Differs from ONNX

| Dimension | ONNX (YOLO/MobileNet) | Transformer |
|-----------|----------------------|-------------|
| **What it does** | Computer vision — detects UI elements from screenshots | NLP/action prediction — predicts next action from OCR text |
| **Input** | Screenshot image (base64 PNG) | OCR text string |
| **Output** | Bounding boxes + class labels | Tool name + coordinates |
| **Training** | Pre-trained (exported from Python) | Trained in Go from your datalog |
| **Runtime** | ONNX Runtime (requires CGO + onnxruntime.dll) | Pure Go (Gorgonia — no CGO needed) |
| **Persistence** | `.onnx` file (downloaded) | `model.gob` (auto-trained) |
| **Purpose** | "What elements are on screen?" | "What should I do next?" |

Both systems work together: ONNX tells the agent what's visible, the transformer tells the agent what to do about it.

## ONNX Runtime DLL

`onnx_download` also pulls a compatible `onnxruntime.dll` from
https://github.com/microsoft/onnxruntime/releases/tag/v1.20.1
to the models directory. The Go library is `github.com/yalue/onnxruntime_go` v1.13.0
(ORT API v20), compatible with ORT 1.20.x.

DLL search order: models dir → working dir → `C:\WINDOWS\System32`.
