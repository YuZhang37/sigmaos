package main

import (
	"os"
	"strconv"
	"time"

	"gonum.org/v1/gonum/mat"

	db "sigmaos/debug"
	"sigmaos/proc"
	"sigmaos/sigmaclnt"
)

func main() {
	if len(os.Args) != 2 {
		db.DFatalf("Usage: %v N\n", os.Args[0])
	}
	m, err := NewMatrixMult(os.Args[1:])
	if err != nil {
		db.DFatalf("%v: error %v", os.Args[0], err)
	}
	m.Work()
}

type MatrixMult struct {
	*sigmaclnt.SigmaClnt
	n  int
	m1 *mat.Dense
	m2 *mat.Dense
	m3 *mat.Dense
}

func NewMatrixMult(args []string) (*MatrixMult, error) {
	pe := proc.GetProcEnv()
	db.DPrintf(db.MATMUL, "NewMatrixMul: %v %v", pe.GetPID(), args)
	m := &MatrixMult{}
	sc, err := sigmaclnt.NewSigmaClnt(pe)
	if err != nil {
		return nil, err
	}
	var error error
	m.SigmaClnt = sc
	m.n, error = strconv.Atoi(args[0])
	if error != nil {
		db.DFatalf("Error parsing N: %v", error)
	}
	m.m1 = matrix(m.n)
	m.m2 = matrix(m.n)
	m.m3 = matrix(m.n)
	return m, nil
}

// Create an n x n matrix.
func matrix(n int) *mat.Dense {
	s := make([]float64, n*n)
	for i := 0; i < n*n; i++ {
		s[i] = float64(i)
	}
	return mat.NewDense(n, n, s)
}

func (m *MatrixMult) doMM() {
	// Multiply m.m1 and m.m2, and place the result in m.m3
	m.m3.Mul(m.m1, m.m2)
}

func (m *MatrixMult) Work() {
	err := m.Started()
	if err != nil {
		db.DFatalf("Started: error %v\n", err)
	}
	start := time.Now()
	db.DPrintf(db.MATMUL, "doMM %v", m.ProcEnv().GetPID())
	m.doMM()
	db.DPrintf(db.MATMUL, "doMM done %v: %v", m.ProcEnv().GetPID(), time.Since(start))
	m.ClntExit(proc.NewStatusInfo(proc.StatusOK, "Latency (us)", time.Since(start).Microseconds()))
	db.DPrintf(db.MATMUL, "Exited %v", m.ProcEnv().GetPID())
}
