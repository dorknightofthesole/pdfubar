package ocr

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/eddiegonzales/pdfubar/internal/job"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

const installHint = `tesseract not found on PATH.
Install it, then re-run:

  brew install tesseract

Or on Debian/Ubuntu:

  sudo apt-get install tesseract-ocr
`

// Run OCRs each page image with tesseract (PDF + optional hOCR) and merges to <stem>.ocr.pdf.
func Run(root string, lang string, writeHOCR bool) error {
	if lang == "" {
		lang = "eng"
	}
	if _, err := exec.LookPath("tesseract"); err != nil {
		return fmt.Errorf("%s", installHint)
	}

	j, err := job.LoadJob(root)
	if err != nil {
		return err
	}

	pages, err := filepath.Glob(filepath.Join(job.PagesPath(root), "*.png"))
	if err != nil {
		return err
	}
	sort.Strings(pages)
	if len(pages) == 0 {
		return fmt.Errorf("no page images in %s — run crop first", job.PagesPath(root))
	}

	ocrDir := filepath.Join(root, "ocr_pages")
	if err := os.MkdirAll(ocrDir, 0o755); err != nil {
		return err
	}

	var pdfParts []string
	for i, page := range pages {
		base := strings.TrimSuffix(filepath.Base(page), filepath.Ext(page))
		outBase := filepath.Join(ocrDir, base)
		part := outBase + ".pdf"
		if _, err := os.Stat(part); err == nil {
			pdfParts = append(pdfParts, part)
			fmt.Printf("ocr %s (%d/%d) [cached]\n", filepath.Base(page), i+1, len(pages))
			continue
		}
		args := []string{page, outBase, "-l", lang, "pdf"}
		if writeHOCR {
			args = append(args, "hocr")
		}
		cmd := exec.Command("tesseract", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("tesseract %s: %w", filepath.Base(page), err)
		}
		if _, err := os.Stat(part); err != nil {
			return fmt.Errorf("expected OCR pdf %s: %w", part, err)
		}
		pdfParts = append(pdfParts, part)
		fmt.Printf("ocr %s (%d/%d)\n", filepath.Base(page), i+1, len(pages))
	}

	out := j.OCRPDFPath(root)
	// Avoid pdfcpu trying to create ~/Library/Application Support/pdfcpu.
	api.DisableConfigDir()
	conf := model.NewDefaultConfiguration()
	if err := api.MergeCreateFile(context.Background(), pdfParts, out, false, conf); err != nil {
		return fmt.Errorf("merge ocr pdfs: %w", err)
	}

	j.Status = "ocr"
	if err := job.SaveJob(root, j); err != nil {
		return err
	}
	fmt.Printf("wrote %s (%d pages)\n", out, len(pdfParts))
	return nil
}
