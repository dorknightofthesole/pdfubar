package detect

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"

	"github.com/eddiegonzales/pdfubar/internal/job"
)

const (
	lowConfidenceThreshold = 0.55
	inkThreshold           = 200 // grayscale below this counts as ink on light paper
)

// Run detects left/right page regions for each spread and writes pages.json + overlays.
func Run(root string) error {
	j, err := job.LoadJob(root)
	if err != nil {
		return err
	}
	if err := job.EnsureLayout(root); err != nil {
		return err
	}

	doc := &job.PagesDoc{Spreads: make([]job.Spread, 0, j.SpreadCount)}
	low := 0

	for i := 0; i < j.SpreadCount; i++ {
		path := job.SpreadPNG(root, i)
		img, err := loadPNG(path)
		if err != nil {
			return err
		}
		b := img.Bounds()
		w, h := b.Dx(), b.Dy()

		gutterX, conf := findGutter(img)
		leftBox := contentBBox(img, 0, gutterX)
		rightBox := contentBBox(img, gutterX, w)
		if leftBox.W == 0 {
			leftBox = job.BBox{X: 0, Y: 0, W: gutterX, H: h}
			conf = math.Min(conf, 0.4)
		}
		if rightBox.W == 0 {
			rightBox = job.BBox{X: gutterX, Y: 0, W: w - gutterX, H: h}
			conf = math.Min(conf, 0.4)
		}

		spread := job.Spread{
			ID:         job.SpreadID(i),
			SourcePage: i,
			Image:      filepath.ToSlash(filepath.Join(job.SpreadsDir, job.SpreadID(i)+".png")),
			Width:      w,
			Height:     h,
			Regions: []job.Region{
				{PageIndex: 0, Role: "left", BBox: leftBox, Confidence: conf, GutterX: gutterX},
				{PageIndex: 1, Role: "right", BBox: rightBox, Confidence: conf, GutterX: gutterX},
			},
		}
		doc.Spreads = append(doc.Spreads, spread)

		overlayPath := job.OverlayPNG(root, i)
		if err := writeOverlay(img, spread, overlayPath); err != nil {
			return err
		}
		if conf < lowConfidenceThreshold {
			low++
			fmt.Printf("low confidence %.2f on %s (gutter x=%d)\n", conf, filepath.Base(path), gutterX)
		} else {
			fmt.Printf("detected %s gutter=%d conf=%.2f\n", filepath.Base(path), gutterX, conf)
		}
	}

	if err := job.SavePages(root, doc); err != nil {
		return err
	}
	j.Status = "detected"
	if err := job.SaveJob(root, j); err != nil {
		return err
	}
	fmt.Printf("wrote %s (%d spreads, %d low-confidence)\n", job.PagesJSONPath(root), len(doc.Spreads), low)
	return nil
}

func loadPNG(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	return img, nil
}

// findGutter returns the vertical split x and a confidence score in [0,1].
func findGutter(img image.Image) (int, float64) {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	proj := make([]float64, w)

	for x := 0; x < w; x++ {
		ink := 0
		for y := 0; y < h; y++ {
			if isInk(img.At(b.Min.X+x, b.Min.Y+y)) {
				ink++
			}
		}
		proj[x] = float64(ink) / float64(h)
	}

	// Search for lowest ink density near the center (bright gutter / spine gap).
	lo := w * 35 / 100
	hi := w * 65 / 100
	if lo >= hi {
		lo, hi = w/3, 2*w/3
	}

	bestX := w / 2
	bestScore := math.MaxFloat64
	var sum, sumSq float64
	n := 0
	for x := lo; x < hi; x++ {
		// Smooth with a small window to ignore thin dark noise.
		s := windowMean(proj, x, 3)
		sum += s
		sumSq += s * s
		n++
		if s < bestScore {
			bestScore = s
			bestX = x
		}
	}

	mid := float64(w) / 2
	distPenalty := math.Abs(float64(bestX)-mid) / (float64(w) * 0.15)
	if distPenalty > 1 {
		distPenalty = 1
	}

	mean := sum / float64(n)
	variance := sumSq/float64(n) - mean*mean
	if variance < 0 {
		variance = 0
	}
	std := math.Sqrt(variance)

	// Stronger contrast between gutter valley and surroundings → higher confidence.
	contrast := 0.0
	if mean > 1e-6 {
		contrast = (mean - bestScore) / (mean + 1e-6)
	}
	conf := clamp01(0.35 + 0.45*contrast + 0.15*(1-distPenalty) + 0.05*clamp01(std*4))
	if bestScore > mean*0.85 {
		// Weak valley — fall back toward geometric center.
		bestX = w / 2
		conf = math.Min(conf, 0.45)
	}
	return bestX, conf
}

func windowMean(proj []float64, x, radius int) float64 {
	sum := 0.0
	n := 0
	for i := x - radius; i <= x+radius; i++ {
		if i >= 0 && i < len(proj) {
			sum += proj[i]
			n++
		}
	}
	if n == 0 {
		return 0
	}
	return sum / float64(n)
}

func contentBBox(img image.Image, x0, x1 int) job.BBox {
	b := img.Bounds()
	h := b.Dy()
	if x1 <= x0 {
		return job.BBox{}
	}

	minX, minY := x1, h
	maxX, maxY := x0-1, -1
	margin := 2

	for y := margin; y < h-margin; y++ {
		for x := x0 + margin; x < x1-margin; x++ {
			if !isInk(img.At(b.Min.X+x, b.Min.Y+y)) {
				continue
			}
			if x < minX {
				minX = x
			}
			if x > maxX {
				maxX = x
			}
			if y < minY {
				minY = y
			}
			if y > maxY {
				maxY = y
			}
		}
	}
	if maxX < minX || maxY < minY {
		return job.BBox{}
	}

	pad := 8
	minX = max(x0, minX-pad)
	minY = max(0, minY-pad)
	maxX = min(x1-1, maxX+pad)
	maxY = min(h-1, maxY+pad)
	return job.BBox{X: minX, Y: minY, W: maxX - minX + 1, H: maxY - minY + 1}
}

func isInk(c color.Color) bool {
	r, g, b, a := c.RGBA()
	if a < 0x8000 {
		return false
	}
	// Average to 8-bit grayscale.
	gray := (r + g + b) / 3 / 256
	return gray < inkThreshold
}

func writeOverlay(src image.Image, spread job.Spread, path string) error {
	b := src.Bounds()
	dst := image.NewRGBA(b)
	draw.Draw(dst, b, src, b.Min, draw.Src)

	colors := []color.RGBA{
		{R: 220, G: 40, B: 40, A: 255},
		{R: 40, G: 120, B: 220, A: 255},
	}
	for i, r := range spread.Regions {
		c := colors[i%len(colors)]
		drawRect(dst, r.BBox, c, 3)
	}
	if len(spread.Regions) > 0 && spread.Regions[0].GutterX > 0 {
		gx := spread.Regions[0].GutterX
		drawVLine(dst, gx, b.Min.Y, b.Max.Y, color.RGBA{R: 40, G: 200, B: 80, A: 255}, 2)
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, dst)
}

func drawRect(img *image.RGBA, box job.BBox, c color.RGBA, thickness int) {
	for t := 0; t < thickness; t++ {
		x0, y0 := box.X+t, box.Y+t
		x1, y1 := box.X+box.W-1-t, box.Y+box.H-1-t
		if x1 <= x0 || y1 <= y0 {
			return
		}
		for x := x0; x <= x1; x++ {
			img.Set(x, y0, c)
			img.Set(x, y1, c)
		}
		for y := y0; y <= y1; y++ {
			img.Set(x0, y, c)
			img.Set(x1, y, c)
		}
	}
}

func drawVLine(img *image.RGBA, x, y0, y1 int, c color.RGBA, thickness int) {
	for t := 0; t < thickness; t++ {
		xx := x + t - thickness/2
		for y := y0; y < y1; y++ {
			img.Set(xx, y, c)
		}
	}
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
