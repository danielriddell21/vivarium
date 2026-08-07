package gui

import (
	"image/color"
	"sort"

	"github.com/danielriddell21/vivarium/internal/sim"
)

const maxLineageBands = 8

var lineagePalette = []color.RGBA{
	{0xe6, 0x6b, 0x6b, 0xff},
	{0x6b, 0xb6, 0xe6, 0xff},
	{0x8f, 0xd1, 0x6b, 0xff},
	{0xe6, 0xc4, 0x5a, 0xff},
	{0xb0, 0x8f, 0xe0, 0xff},
	{0x5a, 0xd1, 0xb0, 0xff},
	{0xe6, 0x8f, 0xc4, 0xff},
	{0xc4, 0xb0, 0x8f, 0xff},
}

var colOther = color.RGBA{0x60, 0x60, 0x60, 0xff}

type lineageView struct {
	topIDs   []int
	colorOf  map[int]int
	maxTotal int
}

func computeLineageView(w *sim.World) *lineageView {
	hist := w.LineageHistory()
	peak := make(map[int]int)
	maxTotal := 1
	for _, s := range hist {
		total := 0
		for id, c := range s {
			if c > peak[id] {
				peak[id] = c
			}
			total += c
		}
		if total > maxTotal {
			maxTotal = total
		}
	}
	ids := make([]int, 0, len(peak))
	for id := range peak {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		if peak[ids[i]] != peak[ids[j]] {
			return peak[ids[i]] > peak[ids[j]]
		}
		return ids[i] < ids[j]
	})
	if len(ids) > maxLineageBands {
		ids = ids[:maxLineageBands]
	}
	colorOf := make(map[int]int, len(ids))
	for i, id := range ids {
		colorOf[id] = i
	}
	return &lineageView{topIDs: ids, colorOf: colorOf, maxTotal: maxTotal}
}

// lineageBodyColor recolours agents by founder line, so the world matches the
// lineage panel's bands.
func lineageBodyColor(lv *lineageView) func(*sim.Agent) (color.RGBA, bool) {
	if lv == nil {
		return nil
	}
	return func(a *sim.Agent) (color.RGBA, bool) {
		if ci, ok := lv.colorOf[a.LineageID]; ok {
			return lineagePalette[ci], true
		}
		return colOther, true
	}
}
