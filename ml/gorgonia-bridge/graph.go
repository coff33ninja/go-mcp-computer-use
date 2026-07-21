package bridge

import (
	"gorgonia.org/gorgonia"
	"gorgonia.org/tensor"
)

// Graph wraps Gorgonia's computational graph for building ML models.
type Graph struct {
	g     *gorgonia.ExprGraph
	nodes map[string]*gorgonia.Node
}

// NewGraph creates a new computational graph wrapper.
func NewGraph() *Graph {
	return &Graph{
		g:     gorgonia.NewGraph(),
		nodes: make(map[string]*gorgonia.Node),
	}
}

// NewInput creates a named input node with the given shape.
func (gr *Graph) NewInput(name string, shape tensor.Shape) *gorgonia.Node {
	dims := len(shape)
	n := gorgonia.NewTensor(gr.g, tensor.Float64, dims,
		gorgonia.WithShape(shape...), gorgonia.WithName(name))
	gr.nodes[name] = n
	return n
}

// NewWeight creates a named weight node initialized with Glorot uniform.
func (gr *Graph) NewWeight(name string, shape tensor.Shape) *gorgonia.Node {
	dims := len(shape)
	n := gorgonia.NewTensor(gr.g, tensor.Float64, dims,
		gorgonia.WithShape(shape...), gorgonia.WithName(name),
		gorgonia.WithInit(gorgonia.GlorotU(1.0)))
	gr.nodes[name] = n
	return n
}

// NewBias creates a named bias node initialized to zeros.
func (gr *Graph) NewBias(name string, size int) *gorgonia.Node {
	n := gorgonia.NewVector(gr.g, tensor.Float64,
		gorgonia.WithShape(size), gorgonia.WithName(name),
		gorgonia.WithInit(gorgonia.Zeroes()))
	gr.nodes[name] = n
	return n
}

// MatMul computes matrix multiplication: a × b
func (gr *Graph) MatMul(a, b *gorgonia.Node) (*gorgonia.Node, error) {
	return gorgonia.Mul(a, b)
}

// Add computes element-wise addition.
func (gr *Graph) Add(a, b *gorgonia.Node) (*gorgonia.Node, error) {
	return gorgonia.Add(a, b)
}

// ReLU computes rectified linear unit activation.
func (gr *Graph) ReLU(x *gorgonia.Node) (*gorgonia.Node, error) {
	return gorgonia.Rectify(x)
}

// Softmax computes softmax activation along the last axis.
func (gr *Graph) Softmax(x *gorgonia.Node) (*gorgonia.Node, error) {
	return gorgonia.SoftMax(x)
}

// MSE computes mean squared error loss between prediction and target.
func (gr *Graph) MSE(pred, target *gorgonia.Node) (*gorgonia.Node, error) {
	diff, err := gorgonia.Sub(pred, target)
	if err != nil {
		return nil, err
	}
	sq, err := gorgonia.Square(diff)
	if err != nil {
		return nil, err
	}
	return gorgonia.Mean(sq)
}

// Grad computes gradients of cost w.r.t. the given parameters.
func (gr *Graph) Grad(cost *gorgonia.Node, params ...*gorgonia.Node) error {
	_, err := gorgonia.Grad(cost, params...)
	return err
}

// Node returns a previously created node by name.
func (gr *Graph) Node(name string) *gorgonia.Node {
	return gr.nodes[name]
}

// RawGraph returns the underlying Gorgonia expression graph.
func (gr *Graph) RawGraph() *gorgonia.ExprGraph {
	return gr.g
}

// TapeMachine creates a tape VM for running the graph.
func TapeMachine(g *gorgonia.ExprGraph) gorgonia.VM {
	return gorgonia.NewTapeMachine(g)
}

// Run executes the graph with the given VM.
func Run(vm gorgonia.VM) error {
	return vm.RunAll()
}

// NewAdamSolver creates an Adam optimizer.
func NewAdamSolver(lr float64) gorgonia.Solver {
	return gorgonia.NewAdamSolver(gorgonia.WithLearnRate(lr))
}

// Let sets the value of a node (used for feeding input data).
func Let(n *gorgonia.Node, val interface{}) error {
	return gorgonia.Let(n, val)
}
