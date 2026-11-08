package crop

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
	xdraw "golang.org/x/image/draw"
)

const (
	targetWidthIn  = 6.0
	targetHeightIn = 9.0
	marginRatio    = 0.04
)

// Run crops detected regions, lightly deskews, and places each page on a
// fixed 6×9" canvas at native resolution (center; downscale only if needed).
func Run(root string) error {
	j, err := job.LoadJob(root)
	if err != nil {
		return err
	}
	pages, err := job.LoadPages(root)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(job.PagesPath(root), 0o755); err != nil {
		return err
	}

	dpi := j.DPI
	if dpi <= 0 {
		dpi = job.DefaultDPI
	}
	targetW := int(math.Round(targetWidthIn * float64(dpi)))
	targetH := int(math.Round(targetHeightIn * float64(dpi)))

	pageNum := 1
	for _, spread := range pages.Spreads {
		srcPath := filepath.Join(root, filepath.FromSlash(spread.Image))
		src, err := loadPNG(srcPath)
		if err != nil {
			return err
		}
		for _, role := range []string{"left", "right"} {
			reg := regionByRole(spread, role)
			if reg == nil {
				continue
			}
			cropped := cropRect(src, reg.BBox)
			deskewed, angle := deskew(cropped)
			normalized, scale := fitOnCanvas(deskewed, targetW, targetH)

			out := job.PagePNG(root, pageNum)
			if err := savePNG(out, normalized); err != nil {
				return err
			}
			scaleNote := "1:1"
			if scale < 1 {
				scaleNote = fmt.Sprintf("downscale %.2f", scale)
			}
			fmt.Printf("wrote %s (from %s %s, deskew %.2f°, %s)\n", filepath.Base(out), spread.ID, role, angle, scaleNote)
			pageNum++
		}
	}

	j.Status = "cropped"
	if err := job.SaveJob(root, j); err != nil {
		return err
	}
	fmt.Printf("wrote %d pages to %s\n", pageNum-1, job.PagesPath(root))
	return nil
}

func regionByRole(s job.Spread, role string) *job.Region {
	for i := range s.Regions {
		if s.Regions[i].Role == role {
			return &s.Regions[i]
		}
	}
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

func savePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func cropRect(src image.Image, box job.BBox) image.Image {
	b := src.Bounds()
	r := image.Rect(box.X, box.Y, box.X+box.W, box.Y+box.H).Intersect(b)
	dst := image.NewRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
	draw.Draw(dst, dst.Bounds(), src, r.Min, draw.Src)
	return dst
}

// deskew estimates a small skew angle via projection profiles on a
// downscaled preview, then applies a single full-resolution rotate.
func deskew(src image.Image) (image.Image, float64) {
	preview := downscaleMax(src, 400)
	bestAngle := 0.0
	bestScore := -1.0
	for a := -3.0; a <= 3.0; a += 0.5 {
		rotated := rotate(preview, a, color.White)
		score := projectionPeakiness(rotated)
		if score > bestScore {
			bestScore = score
			bestAngle = a
		}
	}
	if math.Abs(bestAngle) < 0.25 {
		return src, 0
	}
	return rotate(src, bestAngle, color.White), bestAngle
}

func downscaleMax(src image.Image, maxSide int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= maxSide && h <= maxSide {
		return src
	}
	scale := math.Min(float64(maxSide)/float64(w), float64(maxSide)/float64(h))
	nw := int(math.Round(float64(w) * scale))
	nh := int(math.Round(float64(h) * scale))
	if nw < 1 {
		nw = 1
	}
	if nh < 1 {
		nh = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	xdraw.ApproxBiLinear.Scale(dst, dst.Bounds(), src, b, xdraw.Src, nil)
	return dst
}

func projectionPeakiness(img image.Image) float64 {
	b := img.Bounds()
	h := b.Dy()
	if h < 2 {
		return 0
	}
	rowInk := make([]float64, h)
	for y := 0; y < h; y++ {
		ink := 0
		for x := b.Min.X; x < b.Max.X; x++ {
			if isDark(img.At(x, b.Min.Y+y)) {
				ink++
			}
		}
		rowInk[y] = float64(ink)
	}
	var sumDiff float64
	for y := 1; y < h; y++ {
		d := rowInk[y] - rowInk[y-1]
		sumDiff += d * d
	}
	return sumDiff
}

func rotate(src image.Image, degrees float64, bg color.Color) image.Image {
	if degrees == 0 {
		return src
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	rad := degrees * math.Pi / 180
	cosA := math.Cos(rad)
	sinA := math.Sin(rad)

	// Bounding box of rotated rectangle.
	corners := [][2]float64{
		{0, 0}, {float64(w), 0}, {float64(w), float64(h)}, {0, float64(h)},
	}
	minX, minY := math.MaxFloat64, math.MaxFloat64
	maxX, maxY := -math.MaxFloat64, -math.MaxFloat64
	cx, cy := float64(w)/2, float64(h)/2
	for _, p := range corners {
		x := cx + (p[0]-cx)*cosA - (p[1]-cy)*sinA
		y := cy + (p[0]-cx)*sinA + (p[1]-cy)*cosA
		if x < minX {
			minX = x
		}
		if y < minY {
			minY = y
		}
		if x > maxX {
			maxX = x
		}
		if y > maxY {
			maxY = y
		}
	}
	outW := int(math.Ceil(maxX - minX))
	outH := int(math.Ceil(maxY - minY))
	dst := image.NewRGBA(image.Rect(0, 0, outW, outH))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: bg}, image.Point{}, draw.Src)

	ocx, ocy := float64(outW)/2, float64(outH)/2
	for y := 0; y < outH; y++ {
		for x := 0; x < outW; x++ {
			// Inverse rotate to source.
			dx := float64(x) - ocx
			dy := float64(y) - ocy
			sx := cx + dx*cosA + dy*sinA
			sy := cy - dx*sinA + dy*cosA
			ix, iy := int(math.Round(sx)), int(math.Round(sy))
			if ix >= 0 && iy >= 0 && ix < w && iy < h {
				dst.Set(x, y, src.At(b.Min.X+ix, b.Min.Y+iy))
			}
		}
	}
	return dst
}

// fitOnCanvas centers src on a standard page canvas.
// Never upscales: 1:1 when content fits; downscales only when larger than the
// inner margin box. Returns the canvas image and the scale factor applied.
func fitOnCanvas(src image.Image, canvasW, canvasH int) (image.Image, float64) {
	dst := image.NewRGBA(image.Rect(0, 0, canvasW, canvasH))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)

	marginX := int(float64(canvasW) * marginRatio)
	marginY := int(float64(canvasH) * marginRatio)
	innerW := canvasW - 2*marginX
	innerH := canvasH - 2*marginY
	if innerW < 1 || innerH < 1 {
		innerW, innerH = canvasW, canvasH
		marginX, marginY = 0, 0
	}

	sb := src.Bounds()
	sw, sh := sb.Dx(), sb.Dy()
	scale := math.Min(float64(innerW)/float64(sw), float64(innerH)/float64(sh))
	if scale > 1 {
		scale = 1 // never upscale
	}
	nw := int(math.Round(float64(sw) * scale))
	nh := int(math.Round(float64(sh) * scale))
	if nw < 1 {
		nw = 1
	}
	if nh < 1 {
		nh = 1
	}
	ox := marginX + (innerW-nw)/2
	oy := marginY + (innerH-nh)/2
	target := image.Rect(ox, oy, ox+nw, oy+nh)
	if scale == 1 {
		draw.Draw(dst, target, src, sb.Min, draw.Over)
	} else {
		xdraw.CatmullRom.Scale(dst, target, src, sb, xdraw.Over, nil)
	}
	return dst, scale
}

func isDark(c color.Color) bool {
	r, g, b, a := c.RGBA()
	if a < 0x8000 {
		return false
	}
	gray := (r + g + b) / 3 / 256
	return gray < 200
}
