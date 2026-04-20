package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"itmo-lms/document-service/internal/domain"
	"itmo-lms/pkg/platform"
)

var ErrNotFound = errors.New("document job not found")

type Service struct {
	repo    domain.Repository
	dataDir string
}

func NewService(repo domain.Repository, dataDir string) *Service {
	return &Service{repo: repo, dataDir: dataDir}
}

func (s *Service) Compile(ctx context.Context, req domain.CompileRequest) (domain.DocumentJob, error) {
	if req.Title == "" || len(req.Tasks) == 0 {
		return domain.DocumentJob{}, errors.New("title and tasks are required")
	}
	if req.Format == "" {
		req.Format = "tex"
	}
	jobID := platform.NewID("doc")
	return s.completeCompile(ctx, jobID, req)
}

func (s *Service) completeCompile(ctx context.Context, jobID string, req domain.CompileRequest) (domain.DocumentJob, error) {
	source := BuildLatex(req.Title, req.Tasks)
	storageKey := jobID + ".tex"
	mimeType := "text/x-tex"
	kind := "source"
	raw := []byte(source)
	if strings.EqualFold(req.Format, "pdf") {
		storageKey = jobID + ".pdf"
		mimeType = "application/pdf"
		kind = "rendered"
	}
	if err := os.MkdirAll(s.dataDir, 0o755); err != nil {
		return domain.DocumentJob{}, err
	}
	sourcePath := filepath.Join(s.dataDir, jobID+".tex")
	if err := os.WriteFile(sourcePath, []byte(source), 0o644); err != nil {
		return domain.DocumentJob{}, err
	}
	path := filepath.Join(s.dataDir, storageKey)
	if strings.EqualFold(req.Format, "pdf") {
		if err := compilePDF(ctx, sourcePath, s.dataDir); err != nil {
			return domain.DocumentJob{}, err
		}
		fileRaw, err := os.ReadFile(path)
		if err != nil {
			return domain.DocumentJob{}, err
		}
		raw = fileRaw
	}
	info, _ := os.Stat(path)
	now := time.Now().UTC()
	file := domain.DocumentFile{
		ID:         platform.NewID("file"),
		JobID:      jobID,
		Kind:       kind,
		StorageKey: storageKey,
		MimeType:   mimeType,
		Size:       info.Size(),
		Checksum:   checksum(raw),
		CreatedAt:  now,
	}
	job := domain.DocumentJob{ID: jobID, Format: req.Format, Status: "completed", Files: []domain.DocumentFile{file}, CreatedAt: now, CompletedAt: now}
	if _, ok, err := s.repo.GetJob(ctx, jobID); err == nil && ok {
		return job, s.repo.UpdateJob(ctx, job)
	}
	return job, s.repo.SaveJob(ctx, job)
}

func (s *Service) Get(ctx context.Context, id string) (domain.DocumentJob, error) {
	job, ok, err := s.repo.GetJob(ctx, id)
	if err != nil {
		return domain.DocumentJob{}, err
	}
	if !ok {
		return domain.DocumentJob{}, ErrNotFound
	}
	return job, nil
}

func (s *Service) FilePath(job domain.DocumentJob) string {
	return filepath.Join(s.dataDir, job.Files[0].StorageKey)
}

func BuildLatex(title string, tasks []domain.DocumentTask) string {
	var b strings.Builder
	b.WriteString("\\documentclass[12pt,a4paper]{article}\n")
	b.WriteString("\\usepackage[T2A]{fontenc}\n")
	b.WriteString("\\usepackage[utf8]{inputenc}\n")
	b.WriteString("\\usepackage[russian]{babel}\n")
	b.WriteString("\\usepackage{cmap}\n")
	b.WriteString("\\usepackage{amsmath,amssymb,mathtools,geometry,enumitem}\n")
	b.WriteString("\\geometry{margin=2cm}\n")
	b.WriteString("\\setlength{\\parindent}{0pt}\n")
	b.WriteString("\\setlist[enumerate]{leftmargin=*}\n\\begin{document}\n")
	b.WriteString(fmt.Sprintf("\\section*{%s}\n\\begin{enumerate}\n", escapeLatex(title)))
	for _, task := range tasks {
		b.WriteString(fmt.Sprintf("\\item \\textbf{%s}\\\\\n", escapeLatex(task.Title)))
		b.WriteString(task.LatexBody)
		b.WriteString("\n\\vspace{6mm}\n")
	}
	b.WriteString("\\end{enumerate}\n\\end{document}\n")
	return b.String()
}

func escapeLatex(value string) string {
	replacer := strings.NewReplacer("&", "\\&", "%", "\\%", "$", "\\$", "#", "\\#", "_", "\\_", "{", "\\{", "}", "\\}")
	return replacer.Replace(value)
}

func compilePDF(ctx context.Context, sourcePath, outputDir string) error {
	compiler, err := latexCompiler()
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, compiler,
		"-interaction=nonstopmode",
		"-halt-on-error",
		"-output-directory", outputDir,
		sourcePath,
	)
	raw, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s failed: %w: %s", compiler, err, strings.TrimSpace(string(raw)))
	}
	return nil
}

func latexCompiler() (string, error) {
	for _, name := range []string{"pdflatex", "xelatex"} {
		path, err := exec.LookPath(name)
		if err == nil {
			return path, nil
		}
	}
	return "", errors.New("latex compiler is not installed; expected pdflatex or xelatex")
}

func checksum(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
