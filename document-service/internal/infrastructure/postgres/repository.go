package postgres

import (
	"context"
	"database/sql"

	"itmo-lms/document-service/internal/domain"
	pg "itmo-lms/pkg/postgres"
)

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

func (r *Repository) SaveJob(ctx context.Context, job domain.DocumentJob) error {
	_, err := r.db.ExecContext(ctx, `insert into document_jobs(id, format, status, files_json, error, created_at, completed_at) values ($1,$2,$3,$4,$5,$6,$7)`,
		job.ID, job.Format, job.Status, pg.Marshal(job.Files), job.Error, job.CreatedAt, job.CompletedAt)
	return err
}

func (r *Repository) UpdateJob(ctx context.Context, job domain.DocumentJob) error {
	_, err := r.db.ExecContext(ctx, `update document_jobs set format=$2, status=$3, files_json=$4, error=$5, completed_at=$6 where id=$1`,
		job.ID, job.Format, job.Status, pg.Marshal(job.Files), job.Error, job.CompletedAt)
	return err
}

func (r *Repository) GetJob(ctx context.Context, id string) (domain.DocumentJob, bool, error) {
	row := r.db.QueryRowContext(ctx, `select id, format, status, files_json, error, created_at, completed_at from document_jobs where id=$1`, id)
	var item domain.DocumentJob
	var filesRaw []byte
	err := row.Scan(&item.ID, &item.Format, &item.Status, &filesRaw, &item.Error, &item.CreatedAt, &item.CompletedAt)
	if err == sql.ErrNoRows {
		return domain.DocumentJob{}, false, nil
	}
	if err != nil {
		return domain.DocumentJob{}, false, err
	}
	item.Files = pg.Unmarshal[[]domain.DocumentFile](filesRaw)
	return item, true, nil
}
