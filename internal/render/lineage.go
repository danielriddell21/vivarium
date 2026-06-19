package render

import (
	"fmt"
	"image/color"
	"sort"

	"github.com/danielriddell21/vivarium/internal/sim"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const maxLineageBands = 8 // distinct lineages drawn before the rest become "other"

// lineagePalette colours the tracked lineages. Distinct from the species palette
// so the two views read differently.
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

// lineageView selects which lineages get their own band/colour and how to scale
// the stacked chart.
type lineageView struct {
	topIDs   []int
	colorOf  map[int]int // lineage ID -> palette index (top lineages only)
	maxTotal int         // peak total population across the recorded history
}

// computeLineageView picks the most prominent lineages (by peak abundance) from
// the world's lineage history. It is read-only and does not touch the sim RNG.
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

// drawLineagePanel renders a stacked-area chart of lineage abundance over time in
// the bottom-right, with each prominent lineage as a coloured band and the rest
// lumped into grey "other".
func (g *Game) drawLineagePanel(screen *ebiten.Image) {
	lv := g.lineageView
	const pw, ph = 232.0, 196.0
	px := g.World.W - pw - 6
	py := g.World.H - ph - 6
	drawPanel(screen, px, py, pw, ph)
	drawText(screen, "lineages over time", px+6, py+4, colText)

	hist := g.World.LineageHistory()
	if lv == nil || len(hist) < 2 {
		drawText(screen, "gathering history...", px+6, py+22, colText)
		return
	}

	plotX, plotY := px+6, py+34
	plotW, plotH := pw-12, ph-58
	n := len(hist)
	for c := 0; c < int(plotW); c++ {
		idx := c * (n - 1) / (int(plotW) - 1)
		s := hist[idx]
		x := float32(plotX + float64(c))
		bottom := float32(plotY + plotH)

		var accH float32
		total := 0
		for _, cnt := range s {
			total += cnt
		}
		sumTop := 0
		for _, id := range lv.topIDs {
			cnt := s[id]
			sumTop += cnt
			segH := float32(float64(cnt) / float64(lv.maxTotal) * plotH)
			if segH > 0 {
				vector.DrawFilledRect(screen, x, bottom-accH-segH, 1, segH, lineagePalette[lv.colorOf[id]], false)
				accH += segH
			}
		}
		if otherH := float32(float64(total-sumTop) / float64(lv.maxTotal) * plotH); otherH > 0 {
			vector.DrawFilledRect(screen, x, bottom-accH-otherH, 1, otherH, colOther, false)
		}
	}

	// Caption: how many lineages are alive now vs how many get their own band.
	live := 0
	for _, c := range hist[n-1] {
		if c > 0 {
			live++
		}
	}
	drawText(screen, fmt.Sprintf("%d live lineages; top %d shown, rest grey", live, len(lv.topIDs)), px+6, py+ph-16, colText)
}
