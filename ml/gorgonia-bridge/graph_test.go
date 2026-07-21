package bridge

import (
	"testing"

	"gorgonia.org/tensor"
)

func TestNewGraph_NonNil(t *testing.T) {
	gr := NewGraph()
	if gr == nil {
		t.Fatal("expected non-nil graph")
	}
	if gr.RawGraph() == nil {
		t.Fatal("expected non-nil raw graph")
	}
}

func TestNewInput_CreatesNode(t *testing.T) {
	gr := NewGraph()
	n := gr.NewInput("x", tensor.Shape{2, 3})
	if n == nil {
		t.Fatal("expected non-nil node")
	}
	if gr.Node("x") == nil {
		t.Fatal("expected node to be retrievable by name")
	}
}

func TestNewWeight_CreatesNode(t *testing.T) {
	gr := NewGraph()
	w := gr.NewWeight("w", tensor.Shape{3, 4})
	if w == nil {
		t.Fatal("expected non-nil weight")
	}
}

func TestNewBias_CreatesNode(t *testing.T) {
	gr := NewGraph()
	b := gr.NewBias("b", 5)
	if b == nil {
		t.Fatal("expected non-nil bias")
	}
}

func TestMatMul_ValidShapes(t *testing.T) {
	gr := NewGraph()
	x := gr.NewInput("x", tensor.Shape{1, 3})
	w := gr.NewWeight("w", tensor.Shape{3, 4})
	out, err := gr.MatMul(x, w)
	if err != nil {
		t.Fatalf("MatMul failed: %v", err)
	}
	if out == nil {
		t.Fatal("expected non-nil output")
	}
}

func TestMatMul_IncompatibleShapes(t *testing.T) {
	gr := NewGraph()
	x := gr.NewInput("x", tensor.Shape{1, 3})
	w := gr.NewWeight("w", tensor.Shape{5, 4}) // 3 ≠ 5
	_, err := gr.MatMul(x, w)
	if err == nil {
		t.Error("expected error for incompatible shapes")
	}
}

func TestAdd_SameShape(t *testing.T) {
	gr := NewGraph()
	a := gr.NewInput("a", tensor.Shape{2, 3})
	b := gr.NewInput("b", tensor.Shape{2, 3})
	out, err := gr.Add(a, b)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	if out == nil {
		t.Fatal("expected non-nil output")
	}
}

func TestReLU_CreatesNode(t *testing.T) {
	gr := NewGraph()
	x := gr.NewInput("x", tensor.Shape{2, 3})
	out, err := gr.ReLU(x)
	if err != nil {
		t.Fatalf("ReLU failed: %v", err)
	}
	if out == nil {
		t.Fatal("expected non-nil output")
	}
}

func TestSoftmax_CreatesNode(t *testing.T) {
	gr := NewGraph()
	x := gr.NewInput("x", tensor.Shape{2, 5})
	out, err := gr.Softmax(x)
	if err != nil {
		t.Fatalf("Softmax failed: %v", err)
	}
	if out == nil {
		t.Fatal("expected non-nil output")
	}
}

func TestMSE_CreatesCost(t *testing.T) {
	gr := NewGraph()
	pred := gr.NewInput("pred", tensor.Shape{1, 5})
	target := gr.NewInput("target", tensor.Shape{1, 5})
	cost, err := gr.MSE(pred, target)
	if err != nil {
		t.Fatalf("MSE failed: %v", err)
	}
	if cost == nil {
		t.Fatal("expected non-nil cost")
	}
}

func TestGrad_SimpleGraph(t *testing.T) {
	gr := NewGraph()
	x := gr.NewInput("x", tensor.Shape{1, 3})
	w := gr.NewWeight("w", tensor.Shape{3, 3})
	out, _ := gr.MatMul(x, w)
	cost, _ := gr.MSE(out, x)
	err := gr.Grad(cost, w)
	if err != nil {
		t.Fatalf("Grad failed: %v", err)
	}
}

func TestNode_Nonexistent(t *testing.T) {
	gr := NewGraph()
	if gr.Node("nonexistent") != nil {
		t.Error("expected nil for nonexistent node")
	}
}

func TestTapeMachine_NonNil(t *testing.T) {
	gr := NewGraph()
	x := gr.NewInput("x", tensor.Shape{1, 3})
	w := gr.NewWeight("w", tensor.Shape{3, 3})
	_, _ = gr.MatMul(x, w)
	vm := TapeMachine(gr.RawGraph())
	if vm == nil {
		t.Fatal("expected non-nil tape machine")
	}
}

func TestNewAdamSolver_NonNil(t *testing.T) {
	solver := NewAdamSolver(0.001)
	if solver == nil {
		t.Fatal("expected non-nil solver")
	}
}

func TestRun_SimpleForward(t *testing.T) {
	gr := NewGraph()
	x := gr.NewInput("x", tensor.Shape{1, 3})
	w := gr.NewWeight("w", tensor.Shape{3, 3})
	b := gr.NewBias("b", 3)
	mul, _ := gr.MatMul(x, w)
	out, _ := gr.Add(mul, b)

	// bind input value before running
	xVal := tensor.New(tensor.WithShape(1, 3), tensor.WithBacking([]float64{1, 2, 3}))
	if err := Let(x, xVal); err != nil {
		t.Fatalf("Let failed: %v", err)
	}

	vm := TapeMachine(gr.RawGraph())
	if err := Run(vm); err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	_ = out
}

func TestGraph_NamedNodes(t *testing.T) {
	gr := NewGraph()
	gr.NewInput("a", tensor.Shape{2, 2})
	gr.NewInput("b", tensor.Shape{2, 2})
	gr.NewWeight("w", tensor.Shape{2, 2})
	if gr.Node("a") == nil || gr.Node("b") == nil || gr.Node("w") == nil {
		t.Error("not all nodes retrievable by name")
	}
}

func TestLet_SetsInputValue(t *testing.T) {
	gr := NewGraph()
	x := gr.NewInput("x", tensor.Shape{1, 3})
	val := tensor.New(tensor.WithShape(1, 3), tensor.WithBacking([]float64{1, 2, 3}))
	if err := Let(x, val); err != nil {
		t.Fatalf("Let failed: %v", err)
	}
}
