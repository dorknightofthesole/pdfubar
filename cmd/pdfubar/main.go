package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/eddiegonzales/pdfubar/internal/assemble"
	"github.com/eddiegonzales/pdfubar/internal/crop"
	"github.com/eddiegonzales/pdfubar/internal/detect"
	"github.com/eddiegonzales/pdfubar/internal/extract"
	"github.com/eddiegonzales/pdfubar/internal/job"
	"github.com/eddiegonzales/pdfubar/internal/ocr"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:           "pdfubar",
		Short:         "Split two-up scanned book PDFs into single-page searchable PDFs",
		Long:          "pdfubar (PDF + FUBAR): extract spreads, detect page bounds, crop, assemble, and OCR.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	var (
		outDir    string
		jobDir    string
		dpi       int
		lang      string
		hocr      bool
		skipOCR   bool
	)

	extractCmd := &cobra.Command{
		Use:   "extract INPUT.pdf",
		Short: "Rasterize PDF pages to spreads/*.png",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveOut(outDir, args[0])
			if err != nil {
				return err
			}
			return extract.Run(root, args[0], dpi)
		},
	}
	extractCmd.Flags().StringVar(&outDir, "out", "", "job output directory (default: work/<pdf-stem>)")
	extractCmd.Flags().IntVar(&dpi, "dpi", job.DefaultDPI, "rasterization DPI")

	detectCmd := &cobra.Command{
		Use:   "detect",
		Short: "Detect page regions and write overlays + pages.json",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := requireJob(jobDir)
			if err != nil {
				return err
			}
			return detect.Run(root)
		},
	}
	detectCmd.Flags().StringVar(&jobDir, "job", "", "job directory (required)")
	_ = detectCmd.MarkFlagRequired("job")

	cropCmd := &cobra.Command{
		Use:   "crop",
		Short: "Crop detected regions into ereader-sized page images",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := requireJob(jobDir)
			if err != nil {
				return err
			}
			return crop.Run(root)
		},
	}
	cropCmd.Flags().StringVar(&jobDir, "job", "", "job directory (required)")
	_ = cropCmd.MarkFlagRequired("job")

	assembleCmd := &cobra.Command{
		Use:   "assemble",
		Short: "Build <source-stem>.pdf from page images",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := requireJob(jobDir)
			if err != nil {
				return err
			}
			return assemble.Run(root)
		},
	}
	assembleCmd.Flags().StringVar(&jobDir, "job", "", "job directory (required)")
	_ = assembleCmd.MarkFlagRequired("job")

	ocrCmd := &cobra.Command{
		Use:   "ocr",
		Short: "OCR page images into <source-stem>.ocr.pdf (requires tesseract)",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := requireJob(jobDir)
			if err != nil {
				return err
			}
			return ocr.Run(root, lang, hocr)
		},
	}
	ocrCmd.Flags().StringVar(&jobDir, "job", "", "job directory (required)")
	ocrCmd.Flags().StringVar(&lang, "lang", "eng", "tesseract language(s)")
	ocrCmd.Flags().BoolVar(&hocr, "hocr", false, "also write .hocr sidecars")
	_ = ocrCmd.MarkFlagRequired("job")

	runCmd := &cobra.Command{
		Use:   "run INPUT.pdf",
		Short: "Run extract → detect → crop → assemble → ocr",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveOut(outDir, args[0])
			if err != nil {
				return err
			}
			fmt.Printf("job directory: %s\n", root)
			if err := extract.Run(root, args[0], dpi); err != nil {
				return err
			}
			if err := detect.Run(root); err != nil {
				return err
			}
			if err := crop.Run(root); err != nil {
				return err
			}
			if err := assemble.Run(root); err != nil {
				return err
			}
			if skipOCR {
				fmt.Println("skipping OCR (--skip-ocr)")
				return nil
			}
			return ocr.Run(root, lang, hocr)
		},
	}
	runCmd.Flags().StringVar(&outDir, "out", "", "job output directory (default: work/<pdf-stem>)")
	runCmd.Flags().IntVar(&dpi, "dpi", job.DefaultDPI, "rasterization DPI")
	runCmd.Flags().StringVar(&lang, "lang", "eng", "tesseract language(s)")
	runCmd.Flags().BoolVar(&hocr, "hocr", false, "also write .hocr sidecars")
	runCmd.Flags().BoolVar(&skipOCR, "skip-ocr", false, "stop after assemble")

	rootCmd.AddCommand(extractCmd, detectCmd, cropCmd, assembleCmd, ocrCmd, runCmd)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func requireJob(jobDir string) (string, error) {
	if jobDir == "" {
		return "", fmt.Errorf("--job is required")
	}
	abs, err := filepath.Abs(jobDir)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(filepath.Join(abs, job.JobFileName)); err != nil {
		return "", fmt.Errorf("not a job directory (missing %s): %s", job.JobFileName, abs)
	}
	return abs, nil
}

func resolveOut(outDir, inputPDF string) (string, error) {
	if outDir != "" {
		return filepath.Abs(outDir)
	}
	stem := job.StemFromPath(inputPDF)
	if stem == "book" && strings.TrimSpace(filepath.Base(inputPDF)) == "" {
		stem = fmt.Sprintf("job-%d", time.Now().Unix())
	}
	return filepath.Abs(filepath.Join("work", stem))
}
