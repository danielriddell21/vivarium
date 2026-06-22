package sim

import (
	"runtime"
	"sync"
)

// parallelThinkThreshold is the population below which the think phase runs
// single-threaded: for small worlds the goroutine overhead outweighs the gain.
const parallelThinkThreshold = 256

// thinkAll runs every agent's sense+brain pass (Agent.think). That work is
// read-only on shared state and writes only to per-agent fields, so it is safe to
// run concurrently across a pool of workers sized to the machine. Results are
// deterministic and identical to a sequential pass.
func (w *World) thinkAll(active []*Agent) {
	n := len(active)
	workers := runtime.GOMAXPROCS(0)
	if n < parallelThinkThreshold || workers <= 1 {
		for _, a := range active {
			a.think(w)
		}
		return
	}
	if workers > n {
		workers = n
	}

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
