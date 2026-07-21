package transformer

import (
	"encoding/gob"
	"fmt"
	"math/rand"
	"os"

	"gorgonia.org/gorgonia"
	"gorgonia.org/tensor"
)

type Config struct {
	VocabSize  int
	MaxLen     int
	EmbedDim   int
	NumHeads   int
	NumLayers  int
	FFNDim     int
	CoordDim   int
	OutputDim  int
}

func DefaultConfig() Config {
	return Config{
		VocabSize:  2000,
		MaxLen:     128,
		EmbedDim:   256,
		NumHeads:   4,
		NumLayers:  3,
		FFNDim:     512,
		CoordDim:   7,
		OutputDim:  50,
	}
}

type Model interface {
	Forward(tokens [][]int, coords [][]float64) ([][]float64, error)
	Backward(loss float64, lr float64) error
	BackwardWithTarget(target []float64, lr float64) error
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
	wo         *gorgonia.Node
	ff1W, ff1B *gorgonia.Node
	ff2W, ff2B *gorgonia.Node
}

type transformerModel struct {
	cfg  Config
	g    *gorgonia.ExprGraph
	vm   gorgonia.VM
	sol  gorgonia.Solver

	embInput *gorgonia.Node // [1, embedDim] float64
	coordIn  *gorgonia.Node // [1, coordDim] float64
	targetIn *gorgonia.Node // [1, outputDim] float64

	logits *gorgonia.Node
	cost   *gorgonia.Node

	// embedding table (not in graph — looked up in Go)
	embedTable *tensor.Dense // [vocabSize, embedDim]

	coordProjW *gorgonia.Node
	coordProjB *gorgonia.Node

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

	// coord projection
	m.coordProjW = gorgonia.NewMatrix(g, tensor.Float64, gorgonia.WithShape(cfg.CoordDim, d),
		gorgonia.WithName("coordProjW"), gorgonia.WithInit(gorgonia.GlorotU(1.0)))
	m.coordProjB = gorgonia.NewVector(g, tensor.Float64, gorgonia.WithShape(d),
		gorgonia.WithName("coordProjB"), gorgonia.WithInit(gorgonia.Zeroes()))

	// combine: embInput + coords @ W + b
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

	// layers
	m.layers = make([]layerDef, cfg.NumLayers)
	for i := 0; i < cfg.NumLayers; i++ {
		p := fmt.Sprintf("L%d", i)
		ld := layerDef{}

		ld.wo = gorgonia.NewMatrix(g, tensor.Float64, gorgonia.WithShape(d, d),
			gorgonia.WithName(p+"_wo"), gorgonia.WithInit(gorgonia.GlorotU(1.0)))
		attn, err := gorgonia.Mul(x, ld.wo)
		if err != nil {
			return nil, err
		}
		x, err = gorgonia.Add(x, attn)
		if err != nil {
			return nil, err
		}

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
	addVG(m.headW)
	addVG(m.headB)
	for i := range m.layers {
		addVG(m.layers[i].wo)
		addVG(m.layers[i].ff1W)
		addVG(m.layers[i].ff1B)
		addVG(m.layers[i].ff2W)
		addVG(m.layers[i].ff2B)
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

func (m *transformerModel) Forward(tokens [][]int, coords [][]float64) ([][]float64, error) {
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

func (m *transformerModel) allLearnableNodes() []*gorgonia.Node {
	var nodes []*gorgonia.Node
	nodes = append(nodes, m.coordProjW, m.coordProjB, m.headW, m.headB)
	for i := range m.layers {
		nodes = append(nodes, m.layers[i].wo, m.layers[i].ff1W, m.layers[i].ff1B, m.layers[i].ff2W, m.layers[i].ff2B)
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
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return gob.NewEncoder(f).Encode(params)
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

func init() {
	rand.Seed(42)
}
