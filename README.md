# pdfubar

**PDF + FUBAR** — turn two-up scanned book PDFs into single-page, ereader-friendly, searchable PDFs.

```text
PDF → spreads → detect boxes → crop pages → <name>.pdf → <name>.ocr.pdf
```

Output PDFs are named after the **source file** (e.g. `MyScan.pdf` → `MyScan.pdf` / `MyScan.ocr.pdf` in the job dir).

## Install

Requires [Go](https://go.dev/dl/) 1.22+ to build. [Tesseract](https://github.com/tesseract-ocr/tesseract) is only needed for **ocr** / full **run** (phases through **assemble** work without it). Put `tesseract` on your `PATH` and verify with `tesseract --version`.

### macOS (Homebrew — installs Tesseract automatically)

```bash
# From a clone of this repo:
brew install --HEAD ./Formula/pdfubar.rb

# Or tap this repo:
brew tap dorknightofthesole/pdfubar https://github.com/dorknightofthesole/pdfubar
brew install --HEAD dorknightofthesole/pdfubar/pdfubar
```

`depends_on "tesseract"` pulls Tesseract in automatically. After a tagged release, add `url` / `sha256` to [Formula/pdfubar.rb](Formula/pdfubar.rb) and install without `--HEAD`.

### Linux

Build from source, then install Tesseract from your distro:

```bash
git clone https://github.com/dorknightofthesole/pdfubar.git
cd pdfubar
go build -o bin/pdfubar ./cmd/pdfubar
# or: make build
```

```bash
# Debian / Ubuntu
sudo apt update
sudo apt install tesseract-ocr

# Fedora
sudo dnf install tesseract

# Arch
sudo pacman -S tesseract tesseract-data-eng
```

Optional: [Linuxbrew](https://docs.brew.sh/Homebrew-on-Linux) can use the same formula as macOS (`brew install --HEAD ./Formula/pdfubar.rb`), which will also install Tesseract.

### Windows

1. Install [Go](https://go.dev/dl/) and Git.
2. Build pdfubar in PowerShell or Git Bash:

```powershell
git clone https://github.com/dorknightofthesole/pdfubar.git
cd pdfubar
go build -o bin/pdfubar.exe ./cmd/pdfubar
```

3. Install Tesseract (pick one):

```powershell
winget install UB-Mannheim.TesseractOCR
# or: choco install tesseract
# or: scoop install tesseract
```

If you used the UB Mannheim installer, add `C:\Program Files\Tesseract-OCR` to your **PATH**, then open a new terminal and run `tesseract --version`.

```powershell
.\bin\pdfubar.exe run .\MyScan.pdf
```

There is no first-party Windows installer yet; from-source (or a future GitHub Release binary) is the path for now.

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
