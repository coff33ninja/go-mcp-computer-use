# ML Module Design Document

## Problem

The current prediction engine (`internal/actions/adaptive.go`) is statistical:
- Word-frequency → command mapping (TF-IDF-like)
- SQL-indexed training pairs with word overlap scoring
- Coordinate averaging by token

This works for simple cases but fails when:
- OCR context is novel (no word overlap → zero confidence)
- Sequences matter (actions depend on prior state)
- DPI/scale varies (coordinates don't generalize across monitors)
- No generalization — purely memorized patterns

## Goal

Replace with a small transformer (1-5M parameters) that learns:
1. **OCR → Action**: Given screen text, predict tool + args
2. **Sequence → Context**: Use preceding actions as input features
3. **Spatial Awareness**: DPI-relative coordinate encoding
4. **Online Learning**: Improve from each session without full retraining

## Constraints

- **Single exe**: No Python runtime. Training in Go.
- **User-specific**: Each user trains on their own data
- **Low latency**: Predictions must be <50ms for interactive use
- **No GPU required**: CPU-only training must work (GPU optional via Gorgonia CUDA)
- **Tiny model**: 1-5M params — not billions

## Architecture

```
ml/
├── go.mod                  # standalone module
├── DESIGN.md               # this file
├── tokenizer/
│   ├── tokenizer.go        # interface + BPE/subword implementation
│   └── tokenizer_test.go
├── spatial/
│   ├── encoder.go          # DPI-aware coordinate encoding
│   └── encoder_test.go
├── transformer/
│   ├── attention.go        # multi-head self-attention
│   ├── layer.go            # transformer block (attention + FFN)
│   ├── model.go            # full transformer: embed → layers → predict
│   └── model_test.go
├── dataloader/
│   ├── loader.go           # load training_pairs from SQLite
│   └── loader_test.go
├── online/
│   ├── learner.go          # experience replay + online SGD
│   └── learner_test.go
├── export/
│   ├── serializer.go       # save/load model weights (gob or custom)
│   └── serializer_test.go
├── predict/
│   ├── predictor.go        # inference engine (wraps transformer)
│   └── predictor_test.go
└── gorgonia-bridge/
    ├── graph.go            # Gorgonia graph ops wrapper
    └── graph_test.go
```

## Key Decisions

### 1. Gorgonia as base, custom layers for gaps

Gorgonia provides:
- Autodiff (gradient computation)
- Tensor operations (matmul, reshape, softmax)
- Graph execution (tape machine)
- Adam optimizer
- Optional CUDA via Gorgonia CUDA bindings

What Gorgonia lacks (we implement):
- Layer Normalization (can build from tensor ops)
- Multi-head attention (custom graph)
- Positional encoding (sinusoidal, learned)
- Causal masking

### 2. Tokenizer: Simple BPE over OCR vocabulary

The input vocabulary is small: ~5000 UI words (button names, menu items, labels).
- Start with whitespace tokenization + top-2000 subwords
- No need for GPT-style BPE — keep it simple
- Map to fixed embedding dimension (128-256)

### 3. Model size: ~2M params

```
Embedding:     2000 vocab × 256 dim = 512K params
Position:      128 max_len × 256 dim = 32K params
3 Transformer layers:
  Attention:   256 × 256 × 3 (Q,K,V) × 3 layers = 590K params
  FFN:         256 × 512 × 256 × 3 layers = 393K params
  LayerNorm:   256 × 2 × 6 = 3K params
Output head:   256 × (num_tools + coord dims) ≈ 10K params
Total:         ~1.5M params
```

This fits in a few MB, trains in seconds on CPU, and runs inference in <10ms.

### 4. Coordinate encoding

Instead of raw pixel values, encode coordinates relative to:
- Screen resolution (normalize to 0-1)
- DPI scale factor (multiply by scale)
- Window bounds (relative to window origin)

This lets the model generalize across monitor setups.

### 5. Online learning via experience replay

```
Buffer (circular, 10K samples):
  - Store (context, action, outcome) tuples
  - Reservoir sampling to maintain distribution

Learning loop:
  1. Every N actions, sample batch from buffer
  2. Compute loss on batch
  3. Update weights (SGD with small LR)
  4. Optionally save checkpoint

Batch training (startup):
  1. Load all training_pairs from SQLite
  2. Shuffle, batch (size 32-64)
  3. Train for 5-20 epochs
  4. Save model checkpoint
```

### 6. Export format

Use Go's `encoding/gob` for weight serialization:
- Simple, no dependencies
- Fast serialization/deserialization
- Cross-platform (same binary on all OS)
- Small file size (float32 arrays)

Alternative considered: GGUF (via go-llama.cpp) — too heavy for a 2M param model.

## Integration Plan

Phase 1: Scaffold + interfaces (current)
Phase 2: Implement tokenizer + spatial encoder
Phase 3: Implement transformer layers with Gorgonia
Phase 4: Implement dataloader from SQLite
Phase 5: Batch training pipeline
Phase 6: Online learning with experience replay
Phase 7: Replace adaptive.go with ML predictor
Phase 8: Export/import model checkpoints

## Gorgonia Research Notes

### API Pattern (define-then-run)
```go
g := gorgonia.NewGraph()
x := gorgonia.NewTensor(g, tensor.Float64, 2, tensor.WithShape(1, 784), tensor.WithName("x"))
w0 := gorgonia.NewMatrix(g, tensor.Float64, tensor.WithShape(784, 300), tensor.WithName("w0"), tensor.WithInit(gorgonia.GlorotU(1.0)))
b0 := gorgonia.NewVector(g, tensor.Float64, 300, tensor.WithName("b0"), tensor.WithInit(gorgonia.Zeroes()))
l0 := gorgonia.Must(gorgonia.Add(gorgonia.Must(gorgonia.Mul(x, w0)), b0))
vm := gorgonia.NewTapeMachine(g, gorgonia.BindBidirectional(v))
```

### Available Ops
- MatMul, Mul, Add, Sub, Div
- Softmax, ReLU, Sigmoid, Tanh
- Reshape, Transpose, Sum, Mean
- BatchedMatMul (for multi-head attention)
- Grad() for autodiff

### Key Limitations
- No built-in LayerNorm (we build from Mean + Sub + Sqrt + Div)
- No built-in Dropout (we implement with random mask)
- No built-in positional encoding (we implement sinusoidal)
- CUDA requires separate `gorgonia/cuda` package

### Autodiff Support
```go
cost := gorgonia.Must(gorgonia.Mean(gorgonia.Must(gorgonia.Square(gorgonia.Must(gorgonia.Sub(guess, target)))))
if _, err := gorgonia.Grad(cost, params...); err != nil {
    log.Fatal(err)
}
vm := gorgonia.NewTapeMachine(g, gorgonia.BindBidirectional(params...))
solver := gorgonia.NewAdamSolver(gorgonia.WithLearnRate(0.001))
for i := 0; i < epochs; i++ {
    if err := vm.RunAll(); err != nil { log.Fatal(err) }
    if err := solver.Step(params); err != nil { log.Fatal(err) }
    vm.Reset()
}
```
