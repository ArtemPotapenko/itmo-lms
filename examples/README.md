Здесь лежит демонстрационный артефакт рабочей тетради.

Файлы:
- `workbook-demo.tex` — исходный LaTeX
- `workbook-demo.pdf` — собранный PDF
- `workbook-fragments.json` — входные фрагменты `theory/task/work`
- `generated/workbook-from-fragments.tex` — тетрадь, автоматически собранная из фрагментов
- `generated/workbook-from-fragments.pdf` — собранный PDF из этих же фрагментов

Это пример того, как после загрузки теории и задач выглядит итоговый документ.

Пересобрать артефакты можно так:

```bash
go run ./content-service/cmd/workbook-demo
```
