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
	FromCoordDim int
	OutputDim    int
	ArgDim       int
	WindowDim    int
	SequenceLen  int
	HistoryLen   int
}

func (c Config) SeqSlotDim(numTools int) int {
	return numTools + 2 + c.ArgDim
}

func (c Config) PrimaryDim(numTools int) int {
	return numTools + c.FromCoordDim + 2 + c.ArgDim + c.WindowDim
}

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
		CoordDim:   12,
		OutputDim:  50,
		HistoryLen: 5,
	}
}

type ModelSize string

const (
	SizeSmall  ModelSize = "small"
	SizeMedium ModelSize = "medium"
	SizeLarge  ModelSize = "large"
)

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
	default:
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
	ForwardBackward(target []float64) error
	Step(lr float64) error
	ResetGradients() error
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
	qW, kW, vW *gorgonia.Node
	oW         *gorgonia.Node
	ff1W       *gorgonia.Node
	ff1B       *gorgonia.Node // [MaxLen, FFNDim]
	ff2W       *gorgonia.Node
	ff2B       *gorgonia.Node // [MaxLen, d]
	ln1W       *gorgonia.Node // [MaxLen, d]
	ln1B       *gorgonia.Node // [MaxLen, d]
	ln2W       *gorgonia.Node // [MaxLen, d]
	ln2B       *gorgonia.Node // [MaxLen, d]
}

type transformerModel struct {
	cfg  Config
	g    *gorgonia.ExprGraph
	vm   gorgonia.VM
	sol  gorgonia.Solver

	embInput  *gorgonia.Node // [MaxLen, embedDim]
	coordIn   *gorgonia.Node // [MaxLen, coordDim]
	historyIn *gorgonia.Node // [MaxLen, historyLen*embedDim] (if HistoryLen > 0)
	targetIn  *gorgonia.Node // [1, outputDim]

	logits *gorgonia.Node
	cost   *gorgonia.Node

	embedTable *tensor.Dense

	coordProjW   *gorgonia.Node
	coordProjB   *gorgonia.Node
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
	ml := cfg.MaxLen

	m.embedTable = tensor.New(tensor.WithShape(cfg.VocabSize, d), tensor.Of(tensor.Float64))
	limit := math.Sqrt(6.0 / float64(cfg.VocabSize+d))
	for i := 0; i < cfg.VocabSize*d; i++ {
		m.embedTable.Data().([]float64)[i] = (rand.Float64()*2 - 1) * limit
	}

	m.embInput = gorgonia.NewMatrix(g, tensor.Float64, gorgonia.WithShape(ml, d), gorgonia.WithName("embInput"))
	m.coordIn = gorgonia.NewMatrix(g, tensor.Float64, gorgonia.WithShape(ml, cfg.CoordDim), gorgonia.WithName("coordIn"))
	m.targetIn = gorgonia.NewMatrix(g, tensor.Float64, gorgonia.WithShape(1, cfg.OutputDim), gorgonia.WithName("targetIn"))

	histDim := cfg.HistoryLen * d
	if histDim > 0 {
		m.historyIn = gorgonia.NewMatrix(g, tensor.Float64, gorgonia.WithShape(ml, histDim), gorgonia.WithName("historyIn"))
		m.historyProjW = gorgonia.NewMatrix(g, tensor.Float64, gorgonia.WithShape(histDim, d),
			gorgonia.WithName("historyProjW"), gorgonia.WithInit(gorgonia.GlorotU(1.0)))
		m.historyProjB = gorgonia.NewMatrix(g, tensor.Float64, gorgonia.WithShape(ml, d),
			gorgonia.WithName("historyProjB"), gorgonia.WithInit(gorgonia.Zeroes()))
	}

	m.coordProjW = gorgonia.NewMatrix(g, tensor.Float64, gorgonia.WithShape(cfg.CoordDim, d),
		gorgonia.WithName("coordProjW"), gorgonia.WithInit(gorgonia.GlorotU(1.0)))
	m.coordProjB = gorgonia.NewMatrix(g, tensor.Float64, gorgonia.WithShape(ml, d),
		gorgonia.WithName("coordProjB"), gorgonia.WithInit(gorgonia.Zeroes()))

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

	m.layers = make([]layerDef, cfg.NumLayers)
	for i := 0; i < cfg.NumLayers; i++ {
		p := fmt.Sprintf("L%d", i)
		ld := layerDef{}

		ld.qW = gorgonia.NewMatrix(g, tensor.Float64, gorgonia.WithShape(d, d),
			gorgonia.WithName(p+"_qW"), gorgonia.WithInit(gorgonia.GlorotU(1.0)))
		ld.kW = gorgonia.NewMatrix(g, tensor.Float64, gorgonia.WithShape(d, d),
			gorgonia.WithName(p+"_kW"), gorgonia.WithInit(gorgonia.GlorotU(1.0)))
		ld.vW = gorgonia.NewMatrix(g, tensor.Float64, gorgonia.WithShape(d, d),
			gorgonia.WithName(p+"_vW"), gorgonia.WithInit(gorgonia.GlorotU(1.0)))
		ld.oW = gorgonia.NewMatrix(g, tensor.Float64, gorgonia.WithShape(d, d),
			gorgonia.WithName(p+"_oW"), gorgonia.WithInit(gorgonia.GlorotU(1.0)))

		Q, err := gorgonia.Mul(x, ld.qW)
		if err != nil {
			return nil, fmt.Errorf("layer %d Q: %w", i, err)
		}
		K, err := gorgonia.Mul(x, ld.kW)
		if err != nil {
			return nil, fmt.Errorf("layer %d K: %w", i, err)
		}
		V, err := gorgonia.Mul(x, ld.vW)
		if err != nil {
			return nil, fmt.Errorf("layer %d V: %w", i, err)
		}

		// attention scores: Q @ K^T / sqrt(d)
		KT, err := gorgonia.Transpose(K, 1, 0)
		if err != nil {
			return nil, fmt.Errorf("layer %d K^T: %w", i, err)
		}
		scores, err := gorgonia.Mul(Q, KT)
		if err != nil {
			return nil, fmt.Errorf("layer %d scores: %w", i, err)
		}
		scaleConst := gorgonia.NewConstant(1.0 / math.Sqrt(float64(d)))
		scores, err = gorgonia.HadamardProd(scores, scaleConst)
		if err != nil {
			return nil, fmt.Errorf("layer %d scale: %w", i, err)
		}

		// row-wise softmax: exp / row_sum
		expScores, err := gorgonia.Exp(scores)
		if err != nil {
			return nil, fmt.Errorf("layer %d exp: %w", i, err)
		}
		rowSums, err := gorgonia.Sum(expScores, 1)
		if err != nil {
			return nil, fmt.Errorf("layer %d rowSum: %w", i, err)
		}
		rowSums2d, err := gorgonia.Reshape(rowSums, tensor.Shape{ml, 1})
		if err != nil {
			return nil, fmt.Errorf("layer %d rowSum reshape: %w", i, err)
		}
		onesData := make([]float64, ml)
		for idx := range onesData {
			onesData[idx] = 1.0
		}
		ones := gorgonia.NewConstant(tensor.New(tensor.WithShape(1, ml), tensor.WithBacking(onesData)))
		outer, err := gorgonia.Mul(rowSums2d, ones)
		if err != nil {
			return nil, fmt.Errorf("layer %d outer: %w", i, err)
		}
		attnWeights, err := gorgonia.HadamardDiv(expScores, outer)
		if err != nil {
			return nil, fmt.Errorf("layer %d softmax div: %w", i, err)
		}

		attnOut, err := gorgonia.Mul(attnWeights, V)
		if err != nil {
			return nil, fmt.Errorf("layer %d attn: %w", i, err)
		}

		attnProj, err := gorgonia.Mul(attnOut, ld.oW)
		if err != nil {
			return nil, fmt.Errorf("layer %d O proj: %w", i, err)
		}

		x, err = gorgonia.Add(x, attnProj)
		if err != nil {
			return nil, fmt.Errorf("layer %d residual: %w", i, err)
		}

		ld.ln1W = gorgonia.NewMatrix(g, tensor.Float64, gorgonia.WithShape(ml, d),
			gorgonia.WithName(p+"_ln1W"), gorgonia.WithInit(gorgonia.Ones()))
		ld.ln1B = gorgonia.NewMatrix(g, tensor.Float64, gorgonia.WithShape(ml, d),
			gorgonia.WithName(p+"_ln1B"), gorgonia.WithInit(gorgonia.Zeroes()))
		xScaled, err := gorgonia.HadamardProd(x, ld.ln1W)
		if err != nil {
			xScaled = x
		}
		x, err = gorgonia.Add(xScaled, ld.ln1B)
		if err != nil {
			return nil, fmt.Errorf("layer %d layernorm1: %w", i, err)
		}

		ld.ff1W = gorgonia.NewMatrix(g, tensor.Float64, gorgonia.WithShape(d, cfg.FFNDim),
			gorgonia.WithName(p+"_ff1W"), gorgonia.WithInit(gorgonia.GlorotU(1.0)))
		ld.ff1B = gorgonia.NewMatrix(g, tensor.Float64, gorgonia.WithShape(ml, cfg.FFNDim),
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
		ld.ff2B = gorgonia.NewMatrix(g, tensor.Float64, gorgonia.WithShape(ml, d),
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

		ld.ln2W = gorgonia.NewMatrix(g, tensor.Float64, gorgonia.WithShape(ml, d),
			gorgonia.WithName(p+"_ln2W"), gorgonia.WithInit(gorgonia.Ones()))
		ld.ln2B = gorgonia.NewMatrix(g, tensor.Float64, gorgonia.WithShape(ml, d),
			gorgonia.WithName(p+"_ln2B"), gorgonia.WithInit(gorgonia.Zeroes()))
		xScaled2, err := gorgonia.HadamardProd(x, ld.ln2W)
		if err != nil {
			xScaled2 = x
		}
		x, err = gorgonia.Add(xScaled2, ld.ln2B)
		if err != nil {
			return nil, fmt.Errorf("layer %d layernorm2: %w", i, err)
		}

		m.layers[i] = ld
	}

	// mean-pool: x [MaxLen, d] → mean over axis 0 → [d] → reshape [1, d]
	pooled, err := gorgonia.Mean(x, 0)
	if err != nil {
		return nil, fmt.Errorf("pooling mean: %w", err)
	}
	pooled, err = gorgonia.Reshape(pooled, tensor.Shape{1, d})
	if err != nil {
		return nil, fmt.Errorf("pooling reshape: %w", err)
	}

	m.headW = gorgonia.NewMatrix(g, tensor.Float64, gorgonia.WithShape(d, cfg.OutputDim),
		gorgonia.WithName("headW"), gorgonia.WithInit(gorgonia.GlorotU(1.0)))
	m.headB = gorgonia.NewVector(g, tensor.Float64, gorgonia.WithShape(cfg.OutputDim),
		gorgonia.WithName("headB"), gorgonia.WithInit(gorgonia.Zeroes()))

	m.logits, err = gorgonia.Mul(pooled, m.headW)
	if err != nil {
		return nil, err
	}
	m.logits, err = gorgonia.Add(m.logits, m.headB)
	if err != nil {
		return nil, err
	}

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

func (m *transformerModel) lookupEmbedAll(tokens []int) []float64 {
	d := m.cfg.EmbedDim
	ml := m.cfg.MaxLen
	out := make([]float64, ml*d)
	data := m.embedTable.Data().([]float64)
	for pos := 0; pos < ml; pos++ {
		tok := 0
		if pos < len(tokens) {
			tok = tokens[pos]
		}
		if tok > 0 && tok < m.cfg.VocabSize {
			base := tok * d
			copy(out[pos*d:(pos+1)*d], data[base:base+d])
		}
	}
	return out
}

func (m *transformerModel) lookupEmbed(tokens []int) []float64 {
	d := m.cfg.EmbedDim
	emb := make([]float64, d)
	data := m.embedTable.Data().([]float64)
	count := 0
	for _, tok := range tokens {
		if tok <= 0 || tok >= m.cfg.VocabSize {
			continue
		}
		base := tok * d
		for j := 0; j < d; j++ {
			emb[j] += data[base+j]
		}
		count++
	}
	if count > 1 {
		inv := 1.0 / float64(count)
		for j := 0; j < d; j++ {
			emb[j] *= inv
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

	d := m.cfg.EmbedDim
	ml := m.cfg.MaxLen

	embData := m.lookupEmbedAll(tokens[0])
	embT := tensor.New(tensor.WithShape(ml, d), tensor.WithBacking(embData))
	if err := gorgonia.Let(m.embInput, embT); err != nil {
		return nil, err
	}

	// coords: repeat for each token position
	coordData := make([]float64, ml*m.cfg.CoordDim)
	for i := 0; i < ml; i++ {
		copy(coordData[i*m.cfg.CoordDim:(i+1)*m.cfg.CoordDim], coords[0])
	}
	coordT := tensor.New(tensor.WithShape(ml, m.cfg.CoordDim), tensor.WithBacking(coordData))
	if err := gorgonia.Let(m.coordIn, coordT); err != nil {
		return nil, err
	}

	if m.cfg.HistoryLen > 0 && m.historyIn != nil {
		histEmb := make([]float64, ml*m.cfg.HistoryLen*d)
		if len(history) > 0 {
			for i := 0; i < m.cfg.HistoryLen && i < len(history); i++ {
				actionEmb := m.lookupEmbed(history[i])
				for pos := 0; pos < ml; pos++ {
					base := pos*m.cfg.HistoryLen*d + i*d
					copy(histEmb[base:base+d], actionEmb)
				}
			}
		}
		histT := tensor.New(tensor.WithShape(ml, m.cfg.HistoryLen*d), tensor.WithBacking(histEmb))
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

func (m *transformerModel) Step(lr float64) error {
	clipGradients(m.vgNodes, 1.0)
	if err := m.sol.Step(m.vgNodes); err != nil {
		return err
	}
	m.vm.Reset()
	return nil
}

func clipGradients(vgNodes []gorgonia.ValueGrad, maxNorm float64) {
	var totalNorm2 float64
	for _, vg := range vgNodes {
		grad, err := vg.Grad()
		if err != nil || grad == nil {
			continue
		}
		d, ok := grad.(*tensor.Dense)
		if !ok {
			continue
		}
		data, ok := d.Data().([]float64)
		if !ok {
			continue
		}
		for _, v := range data {
			totalNorm2 += v * v
		}
	}
	totalNorm := math.Sqrt(totalNorm2)
	if totalNorm <= maxNorm {
		return
	}
	scale := maxNorm / totalNorm
	for _, vg := range vgNodes {
		grad, err := vg.Grad()
		if err != nil || grad == nil {
			continue
		}
		d, ok := grad.(*tensor.Dense)
		if !ok {
			continue
		}
		data, ok := d.Data().([]float64)
		if !ok {
			continue
		}
		for i := range data {
			data[i] *= scale
		}
	}
}

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
	embedData := m.embedTable.Data().([]float64)
	params = append(params, embedData...)
	return params
}

func (m *transformerModel) LoadParameters(params []float64) error {
	expected := len(m.Parameters())
	if len(params) != expected {
		return fmt.Errorf("transformer: param count mismatch: got %d, want %d", len(params), expected)
	}

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

func ParamCount(cfg Config) int {
	d := cfg.EmbedDim
	ml := cfg.MaxLen
	count := 0
	count += cfg.VocabSize * d
	count += cfg.CoordDim*d + ml*d
	if cfg.HistoryLen > 0 {
		histDim := cfg.HistoryLen * d
		count += histDim*d + ml*d
	}
	for i := 0; i < cfg.NumLayers; i++ {
		count += 4 * d * d
		count += d*cfg.FFNDim + ml*cfg.FFNDim + cfg.FFNDim*d + ml*d
		count += 2 * (ml*d + ml*d)
	}
	count += d*cfg.OutputDim + cfg.OutputDim
	return count
}

type DebugInfo struct {
	Logits     []float64 `json:"logits"`
	ToolProbs  []float64 `json:"tool_probs"`
	EmbedNorm  float64   `json:"embed_norm"`
	OutputNorm float64   `json:"output_norm"`
	ParamCount int       `json:"param_count"`
}

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

	var embedNorm, outNorm float64
	for _, v := range coords {
		embedNorm += v * v
	}
	for _, v := range out {
		outNorm += v * v
	}

	cfg := Config{}
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
