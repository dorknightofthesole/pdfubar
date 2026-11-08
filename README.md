# pdfubar

**PDF + FUBAR** — turn two-up scanned book PDFs into single-page, ereader-friendly, searchable PDFs.

```text
PDF → spreads → detect boxes → crop pages → <name>.pdf → <name>.ocr.pdf
```

Output PDFs are named after the **source file** (e.g. `MyScan.pdf` → `MyScan.pdf` / `MyScan.ocr.pdf` in the job dir).

## Install

### Homebrew (recommended — installs Tesseract automatically)

Homebrew pulls in [Tesseract](https://github.com/tesseract-ocr/tesseract) via `depends_on "tesseract"`.

```bash
# From a clone of this repo:
brew install --HEAD ./Formula/pdfubar.rb

# Or tap this repo:
brew tap dorknightofthesole/pdfubar https://github.com/dorknightofthesole/pdfubar
brew install --HEAD dorknightofthesole/pdfubar/pdfubar
```

After a tagged release, add `url` / `sha256` to [Formula/pdfubar.rb](Formula/pdfubar.rb) and install without `--HEAD`.

### From source

Requires [Go](https://go.dev/dl/) 1.22+.

```bash
git clone https://github.com/dorknightofthesole/pdfubar.git
cd pdfubar
make build
# binary: ./bin/pdfubar
```

For OCR when not using Homebrew:

```bash
brew install tesseract
```

Phases through **assemble** need no Tesseract. Only **ocr** / full **run** require it on `PATH`.

## Usage

### One shot

```bash
pdfubar run MyScan.pdf
# → work/MyScan/MyScan.pdf
# → work/MyScan/MyScan.ocr.pdf
```

Useful flags:

| Flag | Default | Meaning |
|------|---------|---------|
| `--out DIR` | `work/<stem>` | Job directory |
| `--dpi N` | `300` | Rasterization DPI |
| `--skip-ocr` | off | Stop after assemble (no Tesseract needed) |
| `--lang LANG` | `eng` | Tesseract language(s), e.g. `eng+fra` |
| `--hocr` | off | Also write `.hocr` sidecars under `ocr_pages/` |

```bash
pdfubar run MyScan.pdf --skip-ocr
pdfubar run MyScan.pdf --out work/custom --dpi 300 --lang eng
```

### Phased (inspect between steps)

```bash
pdfubar extract MyScan.pdf --out work/myscan [--dpi 300]
pdfubar detect  --job work/myscan
# open work/myscan/overlays/*.png in Preview — check gutter/boxes
pdfubar crop    --job work/myscan
pdfubar assemble --job work/myscan   # → MyScan.pdf
pdfubar ocr     --job work/myscan [--lang eng] [--hocr]   # → MyScan.ocr.pdf
```

| Command | Input | Output |
|---------|-------|--------|
| `extract` | `INPUT.pdf` | `spreads/*.png`, `job.json` |
| `detect` | `--job` | `pages.json`, `overlays/*.png` |
| `crop` | `--job` | `pages/*.png` (6×9″ @ job DPI, native res, centered, no upscale) |
| `assemble` | `--job` | `<stem>.pdf` |
| `ocr` | `--job` | `<stem>.ocr.pdf` (needs `tesseract`) |
| `run` | `INPUT.pdf` | all of the above |

```bash
pdfubar --help
pdfubar <command> --help
```

## Job layout

```text
work/<stem>/
  job.json
  spreads/0001.png      # two-up page images
  pages.json            # detected bboxes (UI contract later)
  overlays/0001.png     # debug: page boxes + gutter
  pages/0001.png        # single pages
  ocr_pages/            # per-page tesseract PDFs (and optional .hocr)
  <stem>.pdf
  <stem>.ocr.pdf
```

## Sample

Included at `resources/117400931-Qabalah-Qlipoth-and-Goetic-Magic.pdf`:

```bash
make run-sample-no-ocr   # split + assemble only
make run-sample          # full pipeline including OCR
```
