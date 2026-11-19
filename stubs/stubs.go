package stubs

import "uk.ac.bris.cs/gameoflife/util"

var ProcessPart = "GolEngine.ProcessPart"

type Request struct {
	World       [][]bool
	Start, End  int
	ImageWidth  int
	ImageHeight int
}

type Response struct {
	Cells []util.Cell
}

