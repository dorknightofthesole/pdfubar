# Homebrew formula for pdfubar.
#
# Install from this repo (local):
#   brew install --HEAD ./Formula/pdfubar.rb
#
# Or via a tap (recommended for others):
#   brew tap dorknightofthesole/pdfubar https://github.com/dorknightofthesole/pdfubar
#   brew install --HEAD dorknightofthesole/pdfubar/pdfubar
#
# depends_on "tesseract" means Homebrew installs Tesseract automatically
# when you install pdfubar (if it is not already present).

class Pdfubar < Formula
  desc "Split two-up scanned book PDFs into single-page searchable PDFs"
  homepage "https://github.com/dorknightofthesole/pdfubar"
  license "MIT"
  head "https://github.com/dorknightofthesole/pdfubar.git", branch: "main"

  # When you cut a release, add a stable block, e.g.:
  # url "https://github.com/dorknightofthesole/pdfubar/archive/refs/tags/v0.1.0.tar.gz"
  # sha256 "REPLACE_WITH_SHA256"
  # version "0.1.0"

  depends_on "go" => :build
  depends_on "tesseract"

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w"), "./cmd/pdfubar"
  end

  test do
    assert_match "pdfubar", shell_output("#{bin}/pdfubar --help")
  end
end
