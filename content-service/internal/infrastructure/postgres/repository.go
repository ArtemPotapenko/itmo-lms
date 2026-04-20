package postgres

import (
	"context"
	"database/sql"

	"itmo-lms/content-service/internal/domain"
	pg "itmo-lms/pkg/postgres"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateTopic(ctx context.Context, topic domain.Topic) error {
	_, err := r.db.ExecContext(ctx, `insert into topics(id, parent_id, title, order_no, status, created_at) values ($1,$2,$3,$4,$5,$6)`,
		topic.ID, topic.ParentID, topic.Title, topic.Order, topic.Status, topic.CreatedAt)
	return err
}

func (r *Repository) ListTopics(ctx context.Context) ([]domain.Topic, error) {
	rows, err := r.db.QueryContext(ctx, `select id, parent_id, title, order_no, status, created_at from topics order by order_no, title`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.Topic
	for rows.Next() {
		var item domain.Topic
		if err := rows.Scan(&item.ID, &item.ParentID, &item.Title, &item.Order, &item.Status, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) CreateTag(ctx context.Context, tag domain.Tag) error {
	_, err := r.db.ExecContext(ctx, `insert into tags(id, code, name, description, kind, status, created_at) values ($1,$2,$3,$4,$5,$6,$7)`,
		tag.ID, tag.Code, tag.Name, tag.Description, tag.Kind, tag.Status, tag.CreatedAt)
	return err
}

func (r *Repository) ListTags(ctx context.Context) ([]domain.Tag, error) {
	rows, err := r.db.QueryContext(ctx, `select id, code, name, description, kind, status, created_at from tags order by kind, code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.Tag
	for rows.Next() {
		var item domain.Tag
		if err := rows.Scan(&item.ID, &item.Code, &item.Name, &item.Description, &item.Kind, &item.Status, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) GetTag(ctx context.Context, id string) (domain.Tag, bool, error) {
	row := r.db.QueryRowContext(ctx, `select id, code, name, description, kind, status, created_at from tags where id=$1`, id)
	var item domain.Tag
	err := row.Scan(&item.ID, &item.Code, &item.Name, &item.Description, &item.Kind, &item.Status, &item.CreatedAt)
	if err == sql.ErrNoRows {
		return domain.Tag{}, false, nil
	}
	return item, err == nil, err
}

func (r *Repository) CreateTask(ctx context.Context, task domain.Task) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `insert into tasks(id, title, latex_body, topic_ids, tags, difficulty, correct_answer, status, author_id, created_at, updated_at)
		values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		task.ID, task.Title, task.LatexBody, pg.Marshal(task.TopicIDs), pg.Marshal(task.Tags), task.Difficulty, task.CorrectAnswer, task.Status, task.AuthorID, task.CreatedAt, task.UpdatedAt); err != nil {
		return err
	}
	for _, tag := range task.Tags {
		if _, err := tx.ExecContext(ctx, `insert into task_tags(task_id, tag_id, weight) values ($1,$2,$3)`, task.ID, tag.TagID, tag.Weight); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *Repository) ListTasks(ctx context.Context, topicID string) ([]domain.Task, error) {
	rows, err := r.db.QueryContext(ctx, `select id, title, latex_body, topic_ids, tags, difficulty, correct_answer, status, author_id, created_at, updated_at from tasks order by difficulty, created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.Task
	for rows.Next() {
		item, err := scanTask(rows.Scan)
		if err != nil {
			return nil, err
		}
		if topicID == "" || contains(item.TopicIDs, topicID) {
			items = append(items, item)
		}
	}
	return items, rows.Err()
}

func (r *Repository) GetTask(ctx context.Context, id string) (domain.Task, bool, error) {
	row := r.db.QueryRowContext(ctx, `select id, title, latex_body, topic_ids, tags, difficulty, correct_answer, status, author_id, created_at, updated_at from tasks where id=$1`, id)
	item, err := scanTask(row.Scan)
	if err == sql.ErrNoRows {
		return domain.Task{}, false, nil
	}
	if err != nil {
		return domain.Task{}, false, err
	}
	return item, true, r.attachTaskTags(ctx, &item)
}

func (r *Repository) CreateTheory(ctx context.Context, theory domain.Theory) error {
	_, err := r.db.ExecContext(ctx, `insert into theories(id, title, body, summary, topic_ids, status, created_at, updated_at) values ($1,$2,$3,$4,$5,$6,$7,$8)`,
		theory.ID, theory.Title, theory.LatexBody, theory.Summary, pg.Marshal(theory.TopicIDs), theory.Status, theory.CreatedAt, theory.UpdatedAt)
	return err
}

func (r *Repository) ListTheory(ctx context.Context, topicID string) ([]domain.Theory, error) {
	rows, err := r.db.QueryContext(ctx, `select id, title, body, summary, topic_ids, status, created_at, updated_at from theories order by created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.Theory
	for rows.Next() {
		var item domain.Theory
		var topicsRaw []byte
		if err := rows.Scan(&item.ID, &item.Title, &item.LatexBody, &item.Summary, &topicsRaw, &item.Status, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.TopicIDs = pg.Unmarshal[[]string](topicsRaw)
		if topicID == "" || contains(item.TopicIDs, topicID) {
			items = append(items, item)
		}
	}
	return items, rows.Err()
}

func (r *Repository) GetTheory(ctx context.Context, id string) (domain.Theory, bool, error) {
	row := r.db.QueryRowContext(ctx, `select id, title, body, summary, topic_ids, status, created_at, updated_at from theories where id=$1`, id)
	var item domain.Theory
	var topicsRaw []byte
	err := row.Scan(&item.ID, &item.Title, &item.LatexBody, &item.Summary, &topicsRaw, &item.Status, &item.CreatedAt, &item.UpdatedAt)
	if err == sql.ErrNoRows {
		return domain.Theory{}, false, nil
	}
	if err != nil {
		return domain.Theory{}, false, err
	}
	item.TopicIDs = pg.Unmarshal[[]string](topicsRaw)
	return item, true, nil
}

func (r *Repository) CreateWork(ctx context.Context, work domain.WorkTemplate) error {
	taskIDs := make([]string, 0)
	for _, item := range work.Items {
		if item.Kind == "task" {
			taskIDs = append(taskIDs, item.ContentID)
		}
	}
	_, err := r.db.ExecContext(ctx, `insert into work_templates(id, title, task_ids, items_json, status, created_by, created_at) values ($1,$2,$3,$4,$5,$6,$7)`,
		work.ID, work.Title, pg.Marshal(taskIDs), pg.Marshal(work.Items), work.Status, work.CreatedBy, work.CreatedAt)
	return err
}

func (r *Repository) ListWorks(ctx context.Context) ([]domain.WorkTemplate, error) {
	rows, err := r.db.QueryContext(ctx, `select id, title, items_json, status, created_by, created_at from work_templates order by created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.WorkTemplate
	for rows.Next() {
		item, err := scanWork(rows.Scan)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) GetWork(ctx context.Context, id string) (domain.WorkTemplate, bool, error) {
	row := r.db.QueryRowContext(ctx, `select id, title, items_json, status, created_by, created_at from work_templates where id=$1`, id)
	item, err := scanWork(row.Scan)
	if err == sql.ErrNoRows {
		return domain.WorkTemplate{}, false, nil
	}
	return item, err == nil, err
}

func scanTask(scan func(dest ...any) error) (domain.Task, error) {
	var item domain.Task
	var topicRaw, tagsRaw []byte
	err := scan(&item.ID, &item.Title, &item.LatexBody, &topicRaw, &tagsRaw, &item.Difficulty, &item.CorrectAnswer, &item.Status, &item.AuthorID, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return domain.Task{}, err
	}
	item.TopicIDs = pg.Unmarshal[[]string](topicRaw)
	item.Tags = pg.Unmarshal[[]domain.TaskTag](tagsRaw)
	return item, nil
}

func scanWork(scan func(dest ...any) error) (domain.WorkTemplate, error) {
	var item domain.WorkTemplate
	var itemsRaw []byte
	err := scan(&item.ID, &item.Title, &itemsRaw, &item.Status, &item.CreatedBy, &item.CreatedAt)
	if err != nil {
		return domain.WorkTemplate{}, err
	}
	item.Items = pg.Unmarshal[[]domain.WorkItem](itemsRaw)
	return item, nil
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func (r *Repository) attachTaskTags(ctx context.Context, task *domain.Task) error {
	rows, err := r.db.QueryContext(ctx, `
		select tt.tag_id, t.code, t.name, t.kind, tt.weight
		from task_tags tt
		join tags t on t.id = tt.tag_id
		where tt.task_id = $1
		order by t.code
	`, task.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	tags := make([]domain.TaskTag, 0)
	for rows.Next() {
		var tag domain.TaskTag
		if err := rows.Scan(&tag.TagID, &tag.Code, &tag.Name, &tag.Kind, &tag.Weight); err != nil {
			return err
		}
		tags = append(tags, tag)
	}
	task.Tags = tags
	return rows.Err()
}
