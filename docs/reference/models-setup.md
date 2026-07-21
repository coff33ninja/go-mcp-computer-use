# Models & ML Setup

## ONNX Models (Computer Vision)

Run `onnx_download` tool to auto-pull all models to `%APPDATA%\go-mcp-computer-use\models\`.

| Model | Format | Source | Status |
|---|---|---|---|
| YOLO11n | ONNX (pre-exported) | https://github.com/ultralytics/assets/releases/download/v8.3.0/yolo11n.onnx | ✅ auto-download |
| MobileNetV3-small | ONNX | https://huggingface.co/diogoneno/gui-element-classifier/resolve/main/mobilenetv3_small.onnx | ✅ auto-download |

No Python or PyTorch required — all models are pre-exported ONNX.

> **Known incompatibility:** YOLO11n from Ultralytics v8.3.0 uses ONNX opset 22, which requires ONNX Runtime 1.21+. The bundled ORT 1.20.x only supports opsets up to 21. If YOLO detection fails, manually export YOLO11n with `opset=21` or upgrade ORT. MobileNetV3-small works with 1.20.x.

## Transformer Model (Go-native, Action Prediction)

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

### Why a Transformer?

The previous statistical engine (`adaptive.go`) used word-frequency → command mapping (TF-IDF-like) with SQL-indexed training pairs. This works for simple cases but fails when:

- **OCR context is novel** — no word overlap → zero confidence
- **Sequences matter** — actions depend on prior state
- **DPI/scale varies** — coordinates don't generalize across monitors
- **No generalization** — purely memorized patterns, can't extrapolate

The transformer learns OCR→Action mappings, uses preceding actions as context, encodes coordinates relative to screen/DPI/window bounds, and generalizes from limited data.

### Constraints

- **Single exe** — no Python runtime. Training in Go via Gorgonia.
- **User-specific** — each user trains on their own local data
- **Low latency** — predictions <50ms for interactive use
- **CPU-only** — no GPU required (GPU optional via Gorgonia CUDA)
- **Tiny model** — ~14K params, not billions

### Architecture

```
ml/
├── go.mod                      # standalone module
├── transformer/model.go        # Gorgonia-based transformer (FFN + residuals + Adam)
├── predict/predictor.go        # inference engine (softmax tools + sigmoid coords)
├── trainer/trainer.go          # batch training pipeline (Adam, MSE loss)
├── dataloader/sqlite.go        # reads training_pairs from SQLite
├── tokenizer/simple.go         # whitespace + frequency tokenizer
├── spatial/encoder.go          # 7-feature DPI-aware coordinate encoding
├── online/learner.go           # experience replay buffer (circular, 10K samples)
├── export/serializer.go        # gob weight serialization
└── gorgonia-bridge/graph.go    # Gorgonia graph ops wrapper
```

### Key Design Decisions

| Decision | Choice | Why |
|----------|--------|-----|
| **Framework** | Gorgonia | Go-native autodiff, tensor ops, Adam optimizer. No Python dependency. |
| **Tokenizer** | Whitespace + top-2000 subwords | UI vocabulary is small (~5K words). No need for BPE. |
| **Model size** | 64-dim, 2 layers, 2 heads (~14K params) | Trains in seconds on CPU, loads in milliseconds. Fits in a few KB. |
| **Coordinate encoding** | 7-feature normalized: [normX, normY, dpiAdjX, dpiAdjY, relX, relY, isValid] | Generalizes across monitors, DPI scales, and window positions. |
| **Weight format** | Go `encoding/gob` | Simple, fast, no dependencies, cross-platform. |
| **Online learning** | Circular replay buffer (10K samples) | Improves from each session without full retraining. |

### Integration

`ml_bridge.go` wires the transformer into the adaptive engine:

1. On startup: `EnsureAdaptive()` loads `model.gob` if present, otherwise trains from `training_pairs` SQLite table
2. On each tool call: `PredictActions()` tries ML inference first, falls back to statistical engine
3. After each session: new training pairs feed back for next training cycle

### Gorgonia Reference

```go
// Define-then-run pattern
g := gorgonia.NewGraph()
x := gorgonia.NewTensor(g, tensor.Float64, 2, tensor.WithShape(1, 784))
w0 := gorgonia.NewMatrix(g, tensor.Float64, tensor.WithShape(784, 300), tensor.WithInit(gorgonia.GlorotU(1.0)))
l0 := gorgonia.Must(gorgonia.Add(gorgonia.Must(gorgonia.Mul(x, w0)), b0))
vm := gorgonia.NewTapeMachine(g, gorgonia.BindBidirectional(v))

// Autodiff + Adam
cost := gorgonia.Must(gorgonia.Mean(gorgonia.Must(gorgonia.Square(gorgonia.Must(gorgonia.Sub(guess, target))))))
gorgonia.Grad(cost, params...)
solver := gorgonia.NewAdamSolver(gorgonia.WithLearnRate(0.001))
```

Available ops: MatMul, Mul, Add, Sub, Div, Softmax, ReLU, Sigmoid, Tanh, Reshape, Transpose, Sum, Mean, BatchedMatMul, Grad().

Built from tensor ops (not built-in): LayerNorm, Dropout, positional encoding.

## ONNX Runtime DLL

`onnx_download` also pulls a compatible `onnxruntime.dll` from
https://github.com/microsoft/onnxruntime/releases/tag/v1.20.1
to the models directory. The Go library is `github.com/yalue/onnxruntime_go` v1.13.0
(ORT API v20), compatible with ORT 1.20.x.

DLL search order: models dir → working dir → `C:\WINDOWS\System32`.
