"""Convert Salesforce GPA-GUI-Detector (model.pt) to the single-class 'icon'
ONNX consumed by go-mcp-computer-use.

The Go detector expects:
  - input  : (1, 3, 640, 640)  float32, 0-255, RGB (YOLO letterbox preprocess)
  - output : (1, 5, 8400)      float32  (cx, cy, w, h, class0=icon)
  - opset  : 12
  - labels : {0: 'icon'}

Usage:
    uv run export.py [--out DIR] [--repo REPO] [--revision REV]

Defaults:
  --repo     Salesforce/GPA-GUI-Detector
  --revision main (a branch/commit/tag; HF resolves it to a snapshot)
  --out      dist/                     (writes dist/gpa_gui_detector.onnx)
  --out      %APPDATA%\\go-mcp-computer-use\\models  (drop straight into the runtime)

Re-run any time to regenerate from source. The Go binary auto-downloads the
converted ONNX from the latest GitHub release when it is missing, so this script
is only needed to (re)produce the artifact for a release.
"""

from __future__ import annotations

import argparse
import os
import sys
from pathlib import Path


def resolve_model_pt(repo: str, revision: str) -> Path:
    """Locate model.pt, downloading it into the HF cache if necessary."""
    from huggingface_hub import hf_hub_download

    local = hf_hub_download(repo_id=repo, filename="model.pt", revision=revision)
    return Path(local)


def export(pt_path: Path, out_dir: Path) -> Path:
    from ultralytics import YOLO

    model = YOLO(str(pt_path))
    print("loaded model, names:", model.names)
    if model.names != {0: "icon"}:
        raise SystemExit(f"unexpected class layout: {model.names} (expected {{0: 'icon'}})")

    out_dir.mkdir(parents=True, exist_ok=True)
    target = out_dir / "gpa_gui_detector.onnx"

    # ultralytics writes alongside model.pt; then we copy to target. Copy, not
    # rename/replace: the HF cache (model.pt dir) and the output dir can live on
    # different drives, and os.replace fails across volumes on Windows (WinError 17).
    exported = model.export(format="onnx", imgsz=640, opset=12, simplify=False, dynamic=False)
    src = Path(exported)
    if src.resolve() != target.resolve():
        import shutil

        shutil.copy2(src, target)
    print("EXPORTED:", target, target.stat().st_size, "bytes")
    return target


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--out", default="dist", help="output directory for gpa_gui_detector.onnx")
    ap.add_argument("--repo", default="Salesforce/GPA-GUI-Detector")
    ap.add_argument("--revision", default="main", help="HF revision/snapshot (default main)")
    args = ap.parse_args()

    pt = resolve_model_pt(args.repo, args.revision)
    print("model.pt:", pt)
    target = export(pt, Path(args.out))
    print("Done. Copy to the release or the runtime models dir as needed.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
