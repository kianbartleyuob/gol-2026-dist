package main

import (
	"flag"
	"net"
	"net/rpc"

	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

type GolEngine struct{}

func (g *GolEngine) ProcessPart(req stubs.Request, res *stubs.Response) error {
	res.Cells = processPart(req.ImageWidth, req.ImageHeight, req.World, req.Start, req.End)
	return nil
}

// processPart and shouldToggle: same as in the previous server/main.go

func main() {
	port := flag.String("port", "8030", "port to listen on")
	flag.Parse()
	rpc.Register(&GolEngine{})
	ln, err := net.Listen("tcp", ":"+*port)
	if err != nil {
		panic(err)
	}
	defer ln.Close()
	rpc.Accept(ln)
}

// checks each cell in a partition for their next value -> outputs slice of cells that need to be toggled
func processPart(imageW int, imageH int, world [][]bool, start, end int) []util.Cell {
	var cellsToToggle []util.Cell

	for i := start; i < end; i++ {
		x := i % imageW
		y := i / imageW

		aliveNeighbours := 0
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				if dx == 0 && dy == 0 {
					continue
				}

				// Wrap around the edges
				nx := (x + dx + imageW) % imageW
				ny := (y + dy + imageH) % imageH

				if world[ny][nx] {
					aliveNeighbours++
				}
			}
		}

		if shouldToggle(world[y][x], aliveNeighbours) {
			cellsToToggle = append(cellsToToggle, util.Cell{X: x, Y: y})
		}
	}

	return cellsToToggle
}

// checks if should toggle after next turn
func shouldToggle(alive bool, aliveNeighbours int) bool {
	if alive {
		return aliveNeighbours < 2 || aliveNeighbours > 3
	}
	return aliveNeighbours == 3
}

