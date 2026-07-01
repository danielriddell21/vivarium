//go:build ebiten

package gui

import "github.com/hajimehoshi/ebiten/v2"

type camera struct {
	x, y, zoom float64
}

func newCamera() camera { return camera{zoom: 1} }

func (c camera) geoM() ebiten.GeoM {
	var g ebiten.GeoM
	g.Translate(-c.x, -c.y)
	g.Scale(c.zoom, c.zoom)
	return g
}

func (c camera) screenToWorld(sx, sy float64) (float64, float64) {
	return sx/c.zoom + c.x, sy/c.zoom + c.y
}

func (c *camera) zoomAt(factor, sx, sy, worldW, worldH float64) {
	wx, wy := c.screenToWorld(sx, sy)
	c.zoom = clamp(c.zoom*factor, 1, 12)
	// Re-anchor so the same world point stays under the cursor.
	c.x = wx - sx/c.zoom
	c.y = wy - sy/c.zoom
	c.clamp(worldW, worldH)
}

func (c *camera) pan(dx, dy, worldW, worldH float64) {
	c.x += dx / c.zoom
	c.y += dy / c.zoom
	c.clamp(worldW, worldH)
}

func (c *camera) clamp(worldW, worldH float64) {
	maxX := worldW - worldW/c.zoom
	maxY := worldH - worldH/c.zoom
	c.x = clamp(c.x, 0, maxX)
	c.y = clamp(c.y, 0, maxY)
}
