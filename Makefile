.PHONY: build run-sample test tidy clean

BIN := bin/pdfubar
SAMPLE := resources/117400931-Qabalah-Qlipoth-and-Goetic-Magic.pdf
JOB := work/sample

build:
	mkdir -p bin
	go build -o $(BIN) ./cmd/pdfubar

tidy:
	go mod tidy

# Full pipeline including OCR (requires tesseract on PATH)
run-sample: build
	./$(BIN) run $(SAMPLE) --out $(JOB)

# Split/assemble only
run-sample-no-ocr: build
	./$(BIN) run $(SAMPLE) --out $(JOB) --skip-ocr

# Local Homebrew install (pulls tesseract via depends_on)
brew-install:
	brew install --HEAD --formula ./Formula/pdfubar.rb

clean:
	rm -rf work bin
