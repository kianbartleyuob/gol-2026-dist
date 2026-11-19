package gol

import (
	"fmt"
	"net/rpc"
	"sync"

	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

var serverAddr = "127.0.0.1:8030"

type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
}

// Job is used for workers to state partition bounds
type Job struct {
	start int
	end   int
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {

	c.ioCommand <- ioInput

	// select initial world file
	filename := fmt.Sprintf("%dx%d", p.ImageWidth, p.ImageHeight)
	c.ioFilename <- filename

	// record bytes into `world`
	world := make([][]bool, p.ImageHeight)
	for y := range world {
		world[y] = make([]bool, p.ImageWidth)
		for x := range world[y] {
			b := <-c.ioInput
			world[y][x] = b == 255
		}
	}

	turn := 0
	c.events <- StateChange{turn, Executing}

	// separate world into partitions
	totalCells := p.ImageWidth * p.ImageHeight
	baseSize, extra := totalCells/p.Threads, totalCells%p.Threads

	partitions := make([]Job, p.Threads)
	startIndex := 0
	for t := 0; t < p.Threads; t++ {
		size := baseSize
		if t < extra {
			size++
		}
		partitions[t] = Job{start: startIndex, end: startIndex + size}
		startIndex += size
	}

	// Connect to server
	client, err := rpc.Dial("tcp", serverAddr)
	if err != nil {
		panic(err)
	}
	defer client.Close()

	fmt.Println("Connected to server successfully")

	for turn = 0; turn < p.Turns; turn++ {
		jobs := make(chan Job, p.Threads)
		results := make(chan []util.Cell, p.Threads)

		for _, job := range partitions {
			jobs <- job
		}
		close(jobs)

		var wg sync.WaitGroup
		for i := 0; i < p.Threads; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				worker(p, client, jobs, results, world)
			}()
		}

		wg.Wait()
		close(results)

		// toggles cells
		for cells := range results {
			for _, cell := range cells {
				world[cell.Y][cell.X] = !world[cell.Y][cell.X]
			}
		}

		c.events <- TurnComplete{CompletedTurns: turn + 1}
	}

	var alive []util.Cell
	for y := range world {
		for x := range world[y] {
			if world[y][x] {
				alive = append(alive, util.Cell{X: x, Y: y})
			}
		}
	}

	c.events <- FinalTurnComplete{
		CompletedTurns: turn,
		Alive:          alive,
	}

	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}

	close(c.events)
}

func worker(p Params, client *rpc.Client, jobs <-chan Job, results chan<- []util.Cell, world [][]bool) {
	for job := range jobs {
		req := stubs.Request{
			World: world,
			Start: job.start,
			End: job.end,
			ImageWidth: p.ImageWidth,
			ImageHeight: p.ImageHeight,
		}
		var res stubs.Response
		err := client.Call(stubs.ProcessPart, req, &res)
		if err != nil {
			panic(err)
		}
		results <- res.Cells
	}
}

