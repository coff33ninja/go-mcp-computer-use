# GPA-GUI → ONNX export

Converts [Salesforce GPA-GUI-Detector](https://huggingface.co/Salesforce/GPA-GUI-Detector)
(`model.pt`, MIT) into the single-class **`icon`** ONNX that
`go-mcp-computer-use` uses as its element box-proposer in the fused annotation
pipeline.

The detector is only a box proposer: it outputs one class (`icon`) bounding
boxes. The authoritative per-control UI type (`button`, `text_input`, ...) comes
from the separate 15-class MobileNet tier (`mobilenetv3_small.onnx`). See
`internal/actions/onnx.go`.

> **Licensing notice:** The model is owned and licensed by its original authors
> (Salesforce, MIT). This project only ships the format-converted ONNX artifact
> and takes no responsibility for the model's behavior, accuracy, or fitness for
> any purpose. Review and comply with the upstream
> [GPA-GUI-Detector license](https://huggingface.co/Salesforce/GPA-GUI-Detector)
> before use, and you are responsible for any downstream use of its outputs.

## Why re-run this?

The Go binary **auto-downloads** `gpa_gui_detector.onnx` from the latest GitHub
release when it is absent from the models dir, so end users never run this.
You only need it when producing (or regenerating) the artifact that is attached
to a release, or iterating on the conversion itself.

Output requirements (must match `internal/actions/onnx.go`):

- input `(1, 3, 640, 640)` float32 RGB 0-255
- output `(1, 5, 8400)` `(cx, cy, w, h, class0)`
- opset **12**
- single class label `{0: 'icon'}` (`yoloNumClasses = 1`)

## Prerequisites

- [uv](https://docs.astral.sh/uv/) (Python 3.11+). No system Python needed.

## Setup

```powershell
# From this directory
uv sync
```

`uv sync` creates a project-local `.venv` and installs the pinned deps from
`pyproject.toml` (ultralytics, huggingface_hub, onnx). Nothing is installed
globally.

## Convert

```powershell
uv run export.py --out dist
```

- Downloads `model.pt` into HF's cache if needed.
- Exports to ONNX (imgsz=640, opset=12).
- Writes `dist/gpa_gui_detector.onnx`.

To drop it straight into the runtime models dir:

```powershell
uv run export.py --out "$env:APPDATA\go-mcp-computer-use\models"
```

## Verify

```powershell
uv run python -c "from ultralytics import YOLO; m=YOLO('dist/gpa_gui_detector.onnx'); print(m.names)"
# expect: {0: 'icon'}
```

## Ship with a release

The release workflow (`.github/workflows/release.yml`) runs this export in CI
and attaches `gpa_gui_detector.onnx` to each versioned release. The Go binary
fetches it from the always-latest release via:

```
https://github.com/coff33ninja/go-mcp-computer-use/releases/latest/download/gpa_gui_detector.onnx
```

(`releases/latest/download/...` is redirtected by GitHub to the newest
non-draft, non-prerelease release's asset, so the URL never needs a hardcoded
version tag.)
