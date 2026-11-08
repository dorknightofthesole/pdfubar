# pdfubar

**PDF + FUBAR** — turn two-up scanned book PDFs into single-page, ereader-friendly, searchable PDFs.

```text
PDF → spreads → detect boxes → crop pages → book.pdf → book.ocr.pdf
```

## Install

```bash
# Go 1.22+
go install github.com/eddiegonzales/pdfubar/cmd/pdfubar@latest   # once published

# From this repo:
make build
# binary: bin/pdfubar
```

### OCR (optional)

Phases through assemble work with **no** system dependencies beyond the Go binary (MuPDF is embedded via go-fitz).

OCR needs [Tesseract](https://github.com/tesseract-ocr/tesseract):

```bash
brew install tesseract
```

## Usage

```bash
# One shot
pdfubar run book.pdf --out work/mybook

# Split only (no OCR)
pdfubar run book.pdf --out work/mybook --skip-ocr

# Phased
pdfubar extract book.pdf --out work/mybook
pdfubar detect  --job work/mybook   # writes overlays/*.png — open in Preview
pdfubar crop    --job work/mybook
pdfubar assemble --job work/mybook
pdfubar ocr     --job work/mybook   # needs tesseract
```

Inspect `work/mybook/overlays/` after detect to verify green gutter / colored page boxes before cropping.

## Job layout

```text
work/<job>/
  job.json
  spreads/0001.png      # two-up scans
  pages.json            # detected bboxes (UI contract)
  overlays/0001.png     # debug boxes
  pages/0001.png        # single pages on 6×9" @ 300 DPI canvas (native res, centered; no upscale)
  book.pdf
  book.ocr.pdf
```

## Sample

```bash
make run-sample-no-ocr
# or with OCR:
make run-sample
```

Uses `resources/117400931-Qabalah-Qlipoth-and-Goetic-Magic.pdf`.
