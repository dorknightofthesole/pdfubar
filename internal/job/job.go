package job

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	JobFileName   = "job.json"
	PagesFileName = "pages.json"
	SpreadsDir    = "spreads"
	OverlaysDir   = "overlays"
	PagesDir      = "pages"
	DefaultDPI    = 300
)

// Job describes a processing workspace for one input PDF.
type Job struct {
	ID          string    `json:"id"`
	InputPDF    string    `json:"input_pdf"`
	OutputBase  string    `json:"output_base"` // stem of source PDF, used for output filenames
	DPI         int       `json:"dpi"`
	SpreadCount int       `json:"spread_count"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// BBox is [x, y, w, h] in image pixel coordinates.
type BBox struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

// Region is one detected page half within a spread.
type Region struct {
	PageIndex  int     `json:"page_index"`
	Role       string  `json:"role"` // left | right
	BBox       BBox    `json:"bbox"`
	Confidence float64 `json:"confidence"`
	GutterX    int     `json:"gutter_x,omitempty"`
}

// Spread holds detection results for one two-up image.
type Spread struct {
	ID         string   `json:"id"`
	SourcePage int      `json:"source_page"`
	Image      string   `json:"image"`
	Width      int      `json:"width"`
	Height     int      `json:"height"`
	Regions    []Region `json:"regions"`
}

// PagesDoc is the detect↔crop↔UI contract.
type PagesDoc struct {
	Spreads []Spread `json:"spreads"`
}

func Dir(root string) string { return root }

func SpreadsPath(root string) string  { return filepath.Join(root, SpreadsDir) }
func OverlaysPath(root string) string { return filepath.Join(root, OverlaysDir) }
func PagesPath(root string) string    { return filepath.Join(root, PagesDir) }
func JobPath(root string) string      { return filepath.Join(root, JobFileName) }
func PagesJSONPath(root string) string {
	return filepath.Join(root, PagesFileName)
}
// OutputStem is the sanitized basename of the source PDF (no extension).
func (j *Job) OutputStem() string {
	if j != nil && j.OutputBase != "" {
		return j.OutputBase
	}
	if j != nil && j.InputPDF != "" {
		return StemFromPath(j.InputPDF)
	}
	return "book"
}

// AssembledPDFPath is <stem>.pdf inside the job directory.
func (j *Job) AssembledPDFPath(root string) string {
	return filepath.Join(root, j.OutputStem()+".pdf")
}

// OCRPDFPath is <stem>.ocr.pdf inside the job directory.
func (j *Job) OCRPDFPath(root string) string {
	return filepath.Join(root, j.OutputStem()+".ocr.pdf")
}

// StemFromPath returns a filesystem-safe stem from a PDF path.
func StemFromPath(path string) string {
	stem := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	stem = sanitizeStem(stem)
	if stem == "" {
		return "book"
	}
	return stem
}

func sanitizeStem(s string) string {
	s = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		default:
			return '-'
		}
	}, s)
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	return strings.Trim(s, "-")
}

func EnsureLayout(root string) error {
	for _, d := range []string{SpreadsPath(root), OverlaysPath(root), PagesPath(root)} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func New(root, inputPDF string, dpi int) (*Job, error) {
	if dpi <= 0 {
		dpi = DefaultDPI
	}
	if err := EnsureLayout(root); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	j := &Job{
		ID:         filepath.Base(root),
		InputPDF:   inputPDF,
		OutputBase: StemFromPath(inputPDF),
		DPI:        dpi,
		Status:     "created",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := SaveJob(root, j); err != nil {
		return nil, err
	}
	return j, nil
}

func SaveJob(root string, j *Job) error {
	j.UpdatedAt = time.Now().UTC()
	return writeJSON(JobPath(root), j)
}

func LoadJob(root string) (*Job, error) {
	var j Job
	if err := readJSON(JobPath(root), &j); err != nil {
		return nil, fmt.Errorf("load job: %w", err)
	}
	return &j, nil
}

func SavePages(root string, doc *PagesDoc) error {
	return writeJSON(PagesJSONPath(root), doc)
}

func LoadPages(root string) (*PagesDoc, error) {
	var doc PagesDoc
	if err := readJSON(PagesJSONPath(root), &doc); err != nil {
		return nil, fmt.Errorf("load pages.json: %w", err)
	}
	return &doc, nil
}

func SpreadID(index int) string {
	return fmt.Sprintf("%04d", index+1)
}

func SpreadPNG(root string, index int) string {
	return filepath.Join(SpreadsPath(root), SpreadID(index)+".png")
}

func OverlayPNG(root string, index int) string {
	return filepath.Join(OverlaysPath(root), SpreadID(index)+".png")
}

func PagePNG(root string, pageNum int) string {
	return filepath.Join(PagesPath(root), fmt.Sprintf("%04d.png", pageNum))
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func readJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}
