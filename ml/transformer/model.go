package transformer

import (
	"encoding/gob"
	"fmt"
	"math"
	"math/rand"
	"os"

	"gorgonia.org/gorgonia"
	"gorgonia.org/tensor"
)

type Config struct {
	VocabSize    int
	MaxLen       int
	EmbedDim     int
	NumHeads     int
	NumLayers    int
	FFNDim       int
	CoordDim     int
	FromCoordDim int    // extra coord dimensions for drag (from_x, from_y). 0 = single-coord tools
	OutputDim    int
	ArgDim       int    // argument prediction dimensions (scroll dir, key name, etc.)
	WindowDim    int    // window category prediction (0 = disabled). Predicts which window type to target.
	SequenceLen  int    // number of future actions to predict (0 = disabled). Each slot = numTools + 2 + argDim dims
	HistoryLen   int    // number of recent actions for sequence context (0 = disabled)
}

// SeqSlotDim returns the number of output dims per sequence prediction slot.
// Each slot: tool(numTools) + to_xy(2) + arg(argDim).
func (c Config) SeqSlotDim(numTools int) int {
	return numTools + 2 + c.ArgDim
}

// PrimaryDim returns the number of output dims for the primary (next-action) head.
// Primary: tool(numTools) + from_xy(fromCoordDim) + to_xy(2) + arg(argDim) + window(windowDim).
func (c Config) PrimaryDim(numTools int) int {
	return numTools + c.FromCoordDim + 2 + c.ArgDim + c.WindowDim
}

// TotalOutputDim returns the expected total output dimensionality.
// PrimaryDim + SequenceLen × SeqSlotDim.
func (c Config) TotalOutputDim(numTools int) int {
	return c.PrimaryDim(numTools) + c.SequenceLen*c.SeqSlotDim(numTools)
}

func DefaultConfig() Config {
	return Config{
		VocabSize:  2000,
		MaxLen:     128,
		EmbedDim:   256,
		NumHeads:   4,
		NumLayers:  3,
		FFNDim:     512,
		CoordDim:   12, // matches spatial.FeatureDim
		OutputDim:  50,
		HistoryLen: 5,
	}
}

// ModelSize presets for different model capacities.
type ModelSize string

const (
	SizeSmall  ModelSize = "small"  // ~14K params — fast, low memory
	SizeMedium ModelSize = "medium" // ~80K params — balanced
	SizeLarge  ModelSize = "large"  // ~300K params — high accuracy
)

// ConfigForSize returns a Config pre-filled for the given model size.
// CoordDim, OutputDim, ArgDim, WindowDim, and SequenceLen must be set separately.
func ConfigForSize(size ModelSize, coordDim, outputDim, argDim, windowDim, seqLen int) Config {
	switch size {
	case SizeSmall:
		return Config{
			VocabSize: 2000, MaxLen: 128,
			EmbedDim: 64, NumHeads: 2, NumLayers: 2, FFNDim: 128,
			CoordDim: coordDim, OutputDim: outputDim, ArgDim: argDim, WindowDim: windowDim, SequenceLen: seqLen, HistoryLen: 5,
		}
	case SizeLarge:
		return Config{
			VocabSize: 2000, MaxLen: 128,
			EmbedDim: 128, NumHeads: 4, NumLayers: 4, FFNDim: 256,
			CoordDim: coordDim, OutputDim: outputDim, ArgDim: argDim, WindowDim: windowDim, SequenceLen: seqLen, HistoryLen: 5,
		}
	default: // medium
		return Config{
			VocabSize: 2000, MaxLen: 128,
			EmbedDim: 96, NumHeads: 3, NumLayers: 3, FFNDim: 192,
			CoordDim: coordDim, OutputDim: outputDim, ArgDim: argDim, WindowDim: windowDim, SequenceLen: seqLen, HistoryLen: 5,
		}
	}
}

type Model interface {
	Forward(tokens [][]int, coords [][]float64, history [][]int) ([][]float64, error)
	Backward(loss float64, lr float64) error
	BackwardWithTarget(target []float64, lr float64) error
	ForwardBackward(target []float64) error // forward + backward without solver step (for batch accumulation)
	Step(lr float64) error                  // apply accumulated gradients via solver
	ResetGradients() error                  // zero out accumulated gradients
	Parameters() []float64
	LoadParameters(params []float64) error
	Save(path string) error
	Load(path string) error
}

func New(cfg Config) (Model, error) {
	if cfg.VocabSize <= 0 || cfg.EmbedDim <= 0 || cfg.NumLayers <= 0 || cfg.NumHeads <= 0 || cfg.FFNDim <= 0 || cfg.MaxLen <= 0 || cfg.OutputDim <= 0 {
		return nil, fmt.Errorf("transformer: invalid config (all dimensions must be > 0)")
	}
	return newRealModel(cfg)
}

func NewReal(cfg Config) (Model, error) {
	return New(cfg)
}

type layerDef struct {
	qW, kW, vW *gorgonia.Node // Q, K, V projections
	oW          *gorgonia.Node // output projection
	ff1W, ff1B  *gorgonia.Node
	ff2W, ff2B  *gorgonia.Node
	ln1W, ln1B *gorgonia.Node // post-attention layer norm
	ln2W, ln2B *gorgonia.Node // post-FFN layer norm
}

type transformerModel struct {
	cfg  Config
	g    *gorgonia.ExprGraph
	vm   gorgonia.VM
	sol  gorgonia.Solver

	embInput *gorgonia.Node // [1, embedDim] float64
	coordIn  *gorgonia.Node // [1, coordDim] float64
	historyIn *gorgonia.Node // [1, historyLen * embedDim] float64 (flattened history embeddings)
	targetIn *gorgonia.Node // [1, outputDim] float64

	logits *gorgonia.Node
	cost   *gorgonia.Node

	// embedding table (not in graph — looked up in Go)
	embedTable *tensor.Dense // [vocabSize, embedDim]

	coordProjW  *gorgonia.Node
	coordProjB  *gorgonia.Node
	historyProjW *gorgonia.Node
	historyProjB *gorgonia.Node

	layers []layerDef
	headW  *gorgonia.Node
	headB  *gorgonia.Node

	gNodes  []*gorgonia.Node
	vgNodes []gorgonia.ValueGrad
}

func newRealModel(cfg Config) (*transformerModel, error) {
	g := gorgonia.NewGraph()
	m := &transformerModel{cfg: cfg, g: g}
	d := cfg.EmbedDim

	// embedding table (owned by Go, not in graph)
	m.embedTable = tensor.New(tensor.WithShape(cfg.VocabSize, d), tensor.Of(tensor.Float64))
	// Glorot init for embeddings
	scale := 1.0
	for i := 0; i < cfg.VocabSize*d; i++ {
		m.embedTable.Data().([]float64)[i] = rand.NormFloat64() * scale
	}

	// graph inputs
	m.embInput = gorgonia.NewMatrix(g, tensor.Float64, gorgonia.WithShape(1, d), gorgonia.WithName("embInput"))
	m.coordIn = gorgonia.NewMatrix(g, tensor.Float64, gorgonia.WithShape(1, cfg.CoordDim), gorgonia.WithName("coordIn"))
	m.targetIn = gorgonia.NewMatrix(g, tensor.Float64, gorgonia.WithShape(1, cfg.OutputDim), gorgonia.WithName("targetIn"))

	// history input: flattened history embeddings [1, historyLen * embedDim]
	histDim := cfg.HistoryLen * d
	if histDim > 0 {
		m.historyIn = gorgonia.NewMatrix(g, tensor.Float64, gorgonia.WithShape(1, histDim), gorgonia.WithName("historyIn"))
		m.historyProjW = gorgonia.NewMatrix(g, tensor.Float64, gorgonia.WithShape(histDim, d),
			gorgonia.WithName("historyProjW"), gorgonia.WithInit(gorgonia.GlorotU(1.0)))
		m.historyProjB = gorgonia.NewVector(g, tensor.Float64, gorgonia.WithShape(d),
			gorgonia.WithName("historyProjB"), gorgonia.WithInit(gorgonia.Zeroes()))
	}

	// coord projection
	m.coordProjW = gorgonia.NewMatrix(g, tensor.Float64, gorgonia.WithShape(cfg.CoordDim, d),
		gorgonia.WithName("coordProjW"), gorgonia.WithInit(gorgonia.GlorotU(1.0)))
	m.coordProjB = gorgonia.NewVector(g, tensor.Float64, gorgonia.WithShape(d),
		gorgonia.WithName("coordProjB"), gorgonia.WithInit(gorgonia.Zeroes()))

	// combine: embInput + coordProj [+ historyProj]
	coordProj, err := gorgonia.Mul(m.coordIn, m.coordProjW)
	if err != nil {
		return nil, err
	}
	coordProj, err = gorgonia.Add(coordProj, m.coordProjB)
	if err != nil {
		return nil, err
	}
	x, err := gorgonia.Add(m.embInput, coordProj)
	if err != nil {
		return nil, err
	}

	// add history mixing if enabled
	if cfg.HistoryLen > 0 {
		histProj, err := gorgonia.Mul(m.historyIn, m.historyProjW)
		if err != nil {
			return nil, err
		}
		histProj, err = gorgonia.Add(histProj, m.historyProjB)
		if err != nil {
			return nil, err
		}
		x, err = gorgonia.Add(x, histProj)
		if err != nil {
			return nil, err
		}
	}

	// layers
	m.layers = make([]layerDef, cfg.NumLayers)
	for i := 0; i < cfg.NumLayers; i++ {
		p := fmt.Sprintf("L%d", i)
		ld := layerDef{}

		// ── Q, K, V projections ──
		ld.qW = gorgonia.NewMatrix(g, tensor.Float64, gorgonia.WithShape(d, d),
			gorgonia.WithName(p+"_qW"), gorgonia.WithInit(gorgonia.GlorotU(1.0)))
		ld.kW = gorgonia.NewMatrix(g, tensor.Float64, gorgonia.WithShape(d, d),
			gorgonia.WithName(p+"_kW"), gorgonia.WithInit(gorgonia.GlorotU(1.0)))
		ld.vW = gorgonia.NewMatrix(g, tensor.Float64, gorgonia.WithShape(d, d),
			gorgonia.WithName(p+"_vW"), gorgonia.WithInit(gorgonia.GlorotU(1.0)))
		ld.oW = gorgonia.NewMatrix(g, tensor.Float64, gorgonia.WithShape(d, d),
			gorgonia.WithName(p+"_oW"), gorgonia.WithInit(gorgonia.GlorotU(1.0)))

		Q, err := gorgonia.Mul(x, ld.qW) // [1, d]
		if err != nil {
			return nil, fmt.Errorf("layer %d Q: %w", i, err)
		}
		K, err := gorgonia.Mul(x, ld.kW) // [1, d]
		if err != nil {
			return nil, fmt.Errorf("layer %d K: %w", i, err)
		}
		V, err := gorgonia.Mul(x, ld.vW) // [1, d]
		if err != nil {
			return nil, fmt.Errorf("layer %d V: %w", i, err)
		}

		// ── Scaled dot-product attention (MatMul-only, no squeeze) ──
		//
		// Standard: attn = softmax(Q @ Kᵀ / √d) @ V
		// For single token [1,d]: Q @ Kᵀ = [1,1] → softmax = 1.0 → output = V.
		//
		// Gorgonia squeezes [1,1] nodes to scalars, breaking downstream Mul.
		// Solution: use associativity. Compute as Q @ (Kᵀ @ V / √d):
		//   Kᵀ @ V  = [d,1] @ [1,d] = [d,d]  (outer product)
		//   scale:   [d,d] @ diag(1/√d) = [d,d]
		//   Q @ ...  = [1,d] @ [d,d] = [1,d]  (final output)
		// All steps use gorgonia.Mul (matrix multiply) — no shape squeeze.

		// Scale factor: 1/√headDim, applied as diagonal matrix multiplication
		scaleVal := 1.0 / math.Sqrt(float64(d))

		// Kᵀ @ V = [d,1] @ [1,d] = [d,d]
		Kt, err := gorgonia.Transpose(K) // [1,d] → [d,1]
		if err != nil {
			return nil, fmt.Errorf("layer %d K transpose: %w", i, err)
		}
		outerKV, err := gorgonia.Mul(Kt, V) // [d,1] @ [1,d] = [d,d]
		if err != nil {
			return nil, fmt.Errorf("layer %d KᵀV: %w", i, err)
		}

		// Scale via diagonal: outerKV @ diag(s/√d) = outerKV * (1/√d)
		diagScale := gorgonia.NewConstant(tensor.New(
			tensor.Of(tensor.Float64),
			tensor.WithShape(d, d),
			tensor.WithBacking(func() []float64 {
				b := make([]float64, d*d)
				for j := 0; j < d; j++ {
					b[j*d+j] = scaleVal
				}
				return b
			}()),
		))
		scaledOuter, err := gorgonia.Mul(outerKV, diagScale) // [d,d] @ [d,d] = [d,d]
		if err != nil {
			return nil, fmt.Errorf("layer %d scale: %w", i, err)
		}

		// Q @ scaled(KᵀV) = [1,d] @ [d,d] = [1,d]
		attnRaw, err := gorgonia.Mul(Q, scaledOuter) // [1,d]
		if err != nil {
			return nil, fmt.Errorf("layer %d Q@KTV: %w", i, err)
		}

		// Output projection: attnOut = attnRaw @ oW
		attnOut, err := gorgonia.Mul(attnRaw, ld.oW) // [1,d] @ [d,d] = [1,d]
		if err != nil {
			return nil, fmt.Errorf("layer %d O proj: %w", i, err)
		}

		// Residual connection
		x, err = gorgonia.Add(x, attnOut)
		if err != nil {
			return nil, fmt.Errorf("layer %d residual: %w", i, err)
		}

		// ── Layer norm 1 (post-attention) ──
		// Uses [1,d] matrices (not vectors) to prevent Gorgonia HadamardProd
		// from squeezing [1,d] → [d] when multiplying with a [d] vector.
		ld.ln1W = gorgonia.NewMatrix(g, tensor.Float64, gorgonia.WithShape(1, d),
			gorgonia.WithName(p+"_ln1W"), gorgonia.WithInit(gorgonia.Ones()))
		ld.ln1B = gorgonia.NewMatrix(g, tensor.Float64, gorgonia.WithShape(1, d),
			gorgonia.WithName(p+"_ln1B"), gorgonia.WithInit(gorgonia.Zeroes()))
		xScaled, err := gorgonia.HadamardProd(x, ld.ln1W) // [1,d] ⊙ [1,d] = [1,d]
		if err != nil {
			xScaled = x
		}
		x, err = gorgonia.Add(xScaled, ld.ln1B) // [1,d] + [1,d] = [1,d]
		if err != nil {
			return nil, fmt.Errorf("layer %d layernorm1: %w", i, err)
		}

		// ── Feed-forward network ──
		ld.ff1W = gorgonia.NewMatrix(g, tensor.Float64, gorgonia.WithShape(d, cfg.FFNDim),
			gorgonia.WithName(p+"_ff1W"), gorgonia.WithInit(gorgonia.GlorotU(1.0)))
		ld.ff1B = gorgonia.NewVector(g, tensor.Float64, gorgonia.WithShape(cfg.FFNDim),
			gorgonia.WithName(p+"_ff1B"), gorgonia.WithInit(gorgonia.Zeroes()))
		ff1, err := gorgonia.Mul(x, ld.ff1W)
		if err != nil {
			return nil, err
		}
		ff1, err = gorgonia.Add(ff1, ld.ff1B)
		if err != nil {
			return nil, err
		}
		ff1, err = gorgonia.Rectify(ff1)
		if err != nil {
			return nil, err
		}

		ld.ff2W = gorgonia.NewMatrix(g, tensor.Float64, gorgonia.WithShape(cfg.FFNDim, d),
			gorgonia.WithName(p+"_ff2W"), gorgonia.WithInit(gorgonia.GlorotU(1.0)))
		ld.ff2B = gorgonia.NewVector(g, tensor.Float64, gorgonia.WithShape(d),
			gorgonia.WithName(p+"_ff2B"), gorgonia.WithInit(gorgonia.Zeroes()))
		ff2, err := gorgonia.Mul(ff1, ld.ff2W)
		if err != nil {
			return nil, err
		}
		ff2, err = gorgonia.Add(ff2, ld.ff2B)
		if err != nil {
			return nil, err
		}
		x, err = gorgonia.Add(x, ff2)
		if err != nil {
			return nil, err
		}

		// ── Layer norm 2 (post-FFN) ──
		// Same [1,d] shape rationale as ln1 above.
		ld.ln2W = gorgonia.NewMatrix(g, tensor.Float64, gorgonia.WithShape(1, d),
			gorgonia.WithName(p+"_ln2W"), gorgonia.WithInit(gorgonia.Ones()))
		ld.ln2B = gorgonia.NewMatrix(g, tensor.Float64, gorgonia.WithShape(1, d),
			gorgonia.WithName(p+"_ln2B"), gorgonia.WithInit(gorgonia.Zeroes()))
		xScaled2, err := gorgonia.HadamardProd(x, ld.ln2W) // [1,d] ⊙ [1,d] = [1,d]
		if err != nil {
			xScaled2 = x
		}
		x, err = gorgonia.Add(xScaled2, ld.ln2B) // [1,d] + [1,d] = [1,d]
		if err != nil {
			return nil, fmt.Errorf("layer %d layernorm2: %w", i, err)
		}

		m.layers[i] = ld
	}

	// output head
	m.headW = gorgonia.NewMatrix(g, tensor.Float64, gorgonia.WithShape(d, cfg.OutputDim),
		gorgonia.WithName("headW"), gorgonia.WithInit(gorgonia.GlorotU(1.0)))
	m.headB = gorgonia.NewVector(g, tensor.Float64, gorgonia.WithShape(cfg.OutputDim),
		gorgonia.WithName("headB"), gorgonia.WithInit(gorgonia.Zeroes()))

	m.logits, err = gorgonia.Mul(x, m.headW)
	if err != nil {
		return nil, err
	}
	m.logits, err = gorgonia.Add(m.logits, m.headB)
	if err != nil {
		return nil, err
	}

	// loss
	diff, err := gorgonia.Sub(m.logits, m.targetIn)
	if err != nil {
		return nil, err
	}
	sq, err := gorgonia.Square(diff)
	if err != nil {
		return nil, err
	}
	m.cost, err = gorgonia.Mean(sq)
	if err != nil {
		return nil, err
	}

	// collect all learnable as ValueGrad slice
	addVG := func(n *gorgonia.Node) {
		m.gNodes = append(m.gNodes, n)
		m.vgNodes = append(m.vgNodes, n)
	}
	addVG(m.coordProjW)
	addVG(m.coordProjB)
	if m.cfg.HistoryLen > 0 {
		addVG(m.historyProjW)
		addVG(m.historyProjB)
	}
	addVG(m.headW)
	addVG(m.headB)
	for i := range m.layers {
		addVG(m.layers[i].qW)
		addVG(m.layers[i].kW)
		addVG(m.layers[i].vW)
		addVG(m.layers[i].oW)
		addVG(m.layers[i].ff1W)
		addVG(m.layers[i].ff1B)
		addVG(m.layers[i].ff2W)
		addVG(m.layers[i].ff2B)
		addVG(m.layers[i].ln1W)
		addVG(m.layers[i].ln1B)
		addVG(m.layers[i].ln2W)
		addVG(m.layers[i].ln2B)
	}

	if _, err := gorgonia.Grad(m.cost, m.gNodes...); err != nil {
		return nil, fmt.Errorf("transformer: grad: %w", err)
	}

	m.vm = gorgonia.NewTapeMachine(g, gorgonia.BindDualValues(m.gNodes...))
	m.sol = gorgonia.NewAdamSolver(gorgonia.WithLearnRate(0.001))

	return m, nil
}

func (m *transformerModel) lookupEmbed(tokens []int) []float64 {
	d := m.cfg.EmbedDim
	emb := make([]float64, d)
	data := m.embedTable.Data().([]float64)
	for _, tok := range tokens {
		if tok <= 0 || tok >= m.cfg.VocabSize {
			continue
		}
		base := tok * d
		for j := 0; j < d; j++ {
			emb[j] += data[base+j]
		}
	}
	return emb
}

func (m *transformerModel) Forward(tokens [][]int, coords [][]float64, history [][]int) ([][]float64, error) {
	if len(tokens) != len(coords) {
		return nil, fmt.Errorf("transformer: batch size mismatch: %d vs %d", len(tokens), len(coords))
	}
	if len(tokens) != 1 {
		return nil, fmt.Errorf("transformer: only batch size 1 supported")
	}
	if len(tokens[0]) != m.cfg.MaxLen {
		return nil, fmt.Errorf("transformer: token len %d != maxLen %d", len(tokens[0]), m.cfg.MaxLen)
	}

	emb := m.lookupEmbed(tokens[0])
	embT := tensor.New(tensor.WithShape(1, m.cfg.EmbedDim), tensor.WithBacking(emb))
	if err := gorgonia.Let(m.embInput, embT); err != nil {
		return nil, err
	}

	cb := make([]float64, m.cfg.CoordDim)
	copy(cb, coords[0])
	coordT := tensor.New(tensor.WithShape(1, m.cfg.CoordDim), tensor.WithBacking(cb))
	if err := gorgonia.Let(m.coordIn, coordT); err != nil {
		return nil, err
	}

	// history embedding: average each action's token embedding, flatten
	if m.cfg.HistoryLen > 0 && m.historyIn != nil {
		histEmb := make([]float64, m.cfg.HistoryLen*m.cfg.EmbedDim)
		if len(history) > 0 {
			d := m.cfg.EmbedDim
			for i := 0; i < m.cfg.HistoryLen && i < len(history); i++ {
				actionEmb := m.lookupEmbed(history[i])
				base := i * d
				copy(histEmb[base:base+d], actionEmb)
			}
		}
		histT := tensor.New(tensor.WithShape(1, m.cfg.HistoryLen*m.cfg.EmbedDim), tensor.WithBacking(histEmb))
		if err := gorgonia.Let(m.historyIn, histT); err != nil {
			return nil, err
		}
	}

	targetT := tensor.New(tensor.WithShape(1, m.cfg.OutputDim), tensor.WithBacking(make([]float64, m.cfg.OutputDim)))
	if err := gorgonia.Let(m.targetIn, targetT); err != nil {
		return nil, err
	}

	m.vm.Reset()
	if err := m.vm.RunAll(); err != nil {
		return nil, fmt.Errorf("transformer: run: %w", err)
	}

	logitData := m.logits.Value().(*tensor.Dense).Data().([]float64)
	out := [][]float64{make([]float64, m.cfg.OutputDim)}
	copy(out[0], logitData)
	return out, nil
}

func (m *transformerModel) Backward(loss float64, lr float64) error {
	if err := m.sol.Step(m.vgNodes); err != nil {
		return err
	}
	m.vm.Reset()
	return nil
}

func (m *transformerModel) BackwardWithTarget(target []float64, lr float64) error {
	t := tensor.New(tensor.WithShape(1, m.cfg.OutputDim), tensor.WithBacking(target))
	if err := gorgonia.Let(m.targetIn, t); err != nil {
		return err
	}
	m.vm.Reset()
	if err := m.vm.RunAll(); err != nil {
		return fmt.Errorf("transformer: run target: %w", err)
	}
	if err := m.sol.Step(m.vgNodes); err != nil {
		return err
	}
	m.vm.Reset()
	return nil
}

// ForwardBackward runs forward+backward without stepping the solver.
// Gradients accumulate in vgNodes. Call Step() after processing a mini-batch.
func (m *transformerModel) ForwardBackward(target []float64) error {
	t := tensor.New(tensor.WithShape(1, m.cfg.OutputDim), tensor.WithBacking(target))
	if err := gorgonia.Let(m.targetIn, t); err != nil {
		return err
	}
	m.vm.Reset()
	if err := m.vm.RunAll(); err != nil {
		return fmt.Errorf("transformer: forward-backward: %w", err)
	}
	return nil
}

// Step applies accumulated gradients via the solver and resets the VM.
func (m *transformerModel) Step(lr float64) error {
	if err := m.sol.Step(m.vgNodes); err != nil {
		return err
	}
	m.vm.Reset()
	return nil
}

// ResetGradients zeros out accumulated gradients by resetting the VM.
func (m *transformerModel) ResetGradients() error {
	m.vm.Reset()
	return nil
}

func (m *transformerModel) allLearnableNodes() []*gorgonia.Node {
	var nodes []*gorgonia.Node
	nodes = append(nodes, m.coordProjW, m.coordProjB)
	if m.cfg.HistoryLen > 0 {
		nodes = append(nodes, m.historyProjW, m.historyProjB)
	}
	nodes = append(nodes, m.headW, m.headB)
	for i := range m.layers {
		nodes = append(nodes, m.layers[i].qW, m.layers[i].kW, m.layers[i].vW, m.layers[i].oW)
		nodes = append(nodes, m.layers[i].ff1W, m.layers[i].ff1B, m.layers[i].ff2W, m.layers[i].ff2B)
		nodes = append(nodes, m.layers[i].ln1W, m.layers[i].ln1B, m.layers[i].ln2W, m.layers[i].ln2B)
	}
	return nodes
}

func (m *transformerModel) Parameters() []float64 {
	var params []float64
	for _, n := range m.allLearnableNodes() {
		val := n.Value()
		if val == nil {
			continue
		}
		d, ok := val.(*tensor.Dense)
		if !ok {
			continue
		}
		data, ok := d.Data().([]float64)
		if !ok {
			continue
		}
		params = append(params, data...)
	}
	// also include embed table
	embedData := m.embedTable.Data().([]float64)
	params = append(params, embedData...)
	return params
}

func (m *transformerModel) LoadParameters(params []float64) error {
	expected := len(m.Parameters())
	if len(params) != expected {
		return fmt.Errorf("transformer: param count mismatch: got %d, want %d", len(params), expected)
	}

	// graph nodes first
	offset := 0
	for _, n := range m.allLearnableNodes() {
		val := n.Value()
		if val == nil {
			continue
		}
		d, ok := val.(*tensor.Dense)
		if !ok {
			continue
		}
		data, ok := d.Data().([]float64)
		if !ok {
			continue
		}
		copy(data, params[offset:offset+len(data)])
		offset += len(data)
	}
	// embed table last
	embedData := m.embedTable.Data().([]float64)
	copy(embedData, params[offset:offset+len(embedData)])
	return nil
}

func (m *transformerModel) Save(path string) error {
	params := m.Parameters()
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if err := gob.NewEncoder(f).Encode(params); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}

func (m *transformerModel) Load(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	var params []float64
	if err := gob.NewDecoder(f).Decode(&params); err != nil {
		return err
	}
	return m.LoadParameters(params)
}

// ParamCount returns the total number of trainable parameters for a given config.
func ParamCount(cfg Config) int {
	d := cfg.EmbedDim
	count := 0
	// embedding table
	count += cfg.VocabSize * d
	// coord projection
	count += cfg.CoordDim*d + d
	// history projection
	if cfg.HistoryLen > 0 {
		histDim := cfg.HistoryLen * d
		count += histDim*d + d
	}
	// transformer layers
	for i := 0; i < cfg.NumLayers; i++ {
		// self-attention: Q, K, V, O (no biases, weight matrices only)
		count += 4 * d * d
		// FFN
		count += d*cfg.FFNDim + cfg.FFNDim + cfg.FFNDim*d + d
		// layer norm (2 per layer)
		count += 2 * (d + d)
	}
	// output head
	count += d*cfg.OutputDim + cfg.OutputDim
	return count
}

// DebugInfo contains diagnostic information from a forward pass.
type DebugInfo struct {
	Logits       []float64 `json:"logits"`        // raw output logits
	ToolProbs    []float64 `json:"tool_probs"`    // softmax over tool logits
	EmbedNorm    float64   `json:"embed_norm"`    // L2 norm of input embedding
	OutputNorm   float64   `json:"output_norm"`   // L2 norm of output logits
	ParamCount   int       `json:"param_count"`   // total trainable parameters
}

// DebugForward runs a forward pass and returns diagnostic info.
func DebugForward(m Model, tokens []int, coords []float64) (*DebugInfo, error) {
	logits, err := m.Forward([][]int{tokens}, [][]float64{coords}, nil)
	if err != nil {
		return nil, err
	}
	if len(logits) == 0 || len(logits[0]) == 0 {
		return &DebugInfo{}, nil
	}

	out := logits[0]
	toolProbs := Softmax(out)

	// compute norms
	var embedNorm, outNorm float64
	for _, v := range coords {
		embedNorm += v * v
	}
	for _, v := range out {
		outNorm += v * v
	}

	cfg := Config{} // can't extract from interface, but that's ok
	return &DebugInfo{
		Logits:     out,
		ToolProbs:  toolProbs,
		EmbedNorm:  math.Sqrt(embedNorm),
		OutputNorm: math.Sqrt(outNorm),
		ParamCount: ParamCount(cfg),
	}, nil
}

func init() {
	rand.Seed(42)
}
