package sim

import (
	"runtime"
	"sync"
)

const parallelThinkThreshold = 256

func (w *World) thinkAll(active []*Agent) {
	n := len(active)
	workers := runtime.GOMAXPROCS(0)
	if n < parallelThinkThreshold || workers <= 1 {
		for _, a := range active {
			a.think(w)
		}
		return
	}
	workers = min(workers, n)

	chunk := (n + workers - 1) / workers
	var wg sync.WaitGroup
	for start := 0; start < n; start += chunk {
		end := min(start+chunk, n)
		wg.Add(1)
		go func(lo, hi int) {
			defer wg.Done()
			for i := lo; i < hi; i++ {
				active[i].think(w)
			}
		}(start, end)
	}
	wg.Wait()
}
