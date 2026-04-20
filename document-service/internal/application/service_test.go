package application

import (
	"strings"
	"testing"

	"itmo-lms/document-service/internal/domain"
)

func TestBuildLatexSupportsRussianWorkbookContent(t *testing.T) {
	latex := BuildLatex("Рабочая тетрадь", []domain.DocumentTask{
		{ID: "thr_1", Title: "Теория. Дискриминант", LatexBody: "D=b^2-4ac"},
		{ID: "tsk_1", Title: "Задача 1", LatexBody: "x^2-5x+6=0"},
	})

	required := []string{
		"\\usepackage[T2A]{fontenc}",
		"\\usepackage[utf8]{inputenc}",
		"\\usepackage[russian]{babel}",
		"\\usepackage{cmap}",
		"\\section*{Рабочая тетрадь}",
		"\\textbf{Теория. Дискриминант}",
		"\\textbf{Задача 1}",
	}
	for _, item := range required {
		if !strings.Contains(latex, item) {
			t.Fatalf("latex missing %q:\n%s", item, latex)
		}
	}
}
