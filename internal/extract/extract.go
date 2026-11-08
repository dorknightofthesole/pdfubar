package extract

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"

	"github.com/eddiegonzales/pdfubar/internal/job"
	"github.com/gen2brain/go-fitz"
)

// Run rasterizes each PDF page to spreads/*.png and updates job.json.
func Run(root, inputPDF string, dpi int) error {
	absIn, err := filepath.Abs(inputPDF)
	if err != nil {
		return err
	}
	if _, err := os.Stat(absIn); err != nil {
		return fmt.Errorf("input pdf: %w", err)
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}

	j, err := job.New(root, absIn, dpi)
	if err != nil {
		return err
	}

	doc, err := fitz.New(absIn)
	if err != nil {
		return fmt.Errorf("open pdf: %w", err)
	}
	defer doc.Close()

	n := doc.NumPage()
	if n == 0 {
		return fmt.Errorf("pdf has no pages")
	}

	for i := 0; i < n; i++ {
		img, err := doc.ImageDPI(i, float64(j.DPI))
		if err != nil {
			return fmt.Errorf("render page %d: %w", i+1, err)
		}
		out := job.SpreadPNG(root, i)
		f, err := os.Create(out)
		if err != nil {
			return err
		}
		if err := png.Encode(f, img); err != nil {
			f.Close()
			return fmt.Errorf("encode %s: %w", out, err)
		}
		if err := f.Close(); err != nil {
			return err
		}
		fmt.Printf("extracted spread %s (%d/%d)\n", filepath.Base(out), i+1, n)
	}

	j.SpreadCount = n
	j.Status = "extracted"
	if err := job.SaveJob(root, j); err != nil {
		return err
	}
	fmt.Printf("wrote %d spreads to %s\n", n, job.SpreadsPath(root))
	return nil
}
