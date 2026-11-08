package assemble

import (
	"fmt"
	"path/filepath"
	"sort"

	"codeberg.org/go-pdf/fpdf"
	"github.com/eddiegonzales/pdfubar/internal/job"
)

const (
	pageWidthPt  = 6.0 * 72.0 // 6 inches
	pageHeightPt = 9.0 * 72.0 // 9 inches
)

// Run builds <source-stem>.pdf from pages/*.png (one image per page).
func Run(root string) error {
	j, err := job.LoadJob(root)
	if err != nil {
		return err
	}

	files, err := filepath.Glob(filepath.Join(job.PagesPath(root), "*.png"))
	if err != nil {
		return err
	}
	sort.Strings(files)
	if len(files) == 0 {
		return fmt.Errorf("no page images in %s — run crop first", job.PagesPath(root))
	}

	pdf := fpdf.NewCustom(&fpdf.InitType{
		UnitStr: "pt",
		Size:    fpdf.SizeType{Wd: pageWidthPt, Ht: pageHeightPt},
	})
	pdf.SetAutoPageBreak(false, 0)

	for _, path := range files {
		pdf.AddPage()
		opt := fpdf.ImageOptions{ImageType: "PNG", ReadDpi: false}
		pdf.RegisterImageOptions(path, opt)
		pdf.ImageOptions(path, 0, 0, pageWidthPt, pageHeightPt, false, opt, 0, "")
		if pdf.Error() != nil {
			return fmt.Errorf("add %s: %w", filepath.Base(path), pdf.Error())
		}
		fmt.Printf("assembled %s\n", filepath.Base(path))
	}

	out := j.AssembledPDFPath(root)
	if err := pdf.OutputFileAndClose(out); err != nil {
		return fmt.Errorf("write %s: %w", out, err)
	}

	j.Status = "assembled"
	if err := job.SaveJob(root, j); err != nil {
		return err
	}
	fmt.Printf("wrote %s (%d pages)\n", out, len(files))
	return nil
}
