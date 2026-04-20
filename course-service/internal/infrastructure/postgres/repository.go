package postgres

import (
	"context"
	"database/sql"
	"time"

	"itmo-lms/course-service/internal/domain"
	pg "itmo-lms/pkg/postgres"
)

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

func (r *Repository) CreateCourse(ctx context.Context, course domain.Course) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `insert into courses(id, title, owner_id, status, created_at) values ($1,$2,$3,$4,$5)`,
		course.ID, course.Title, course.OwnerID, course.Status, course.CreatedAt); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `insert into course_members(course_id, user_id, role) values ($1,$2,'teacher') on conflict do nothing`, course.ID, course.OwnerID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *Repository) CourseExists(ctx context.Context, id string) (bool, error) {
	return exists(ctx, r.db, `select exists(select 1 from courses where id=$1)`, id)
}

func (r *Repository) ListCourses(ctx context.Context) ([]domain.Course, error) {
	rows, err := r.db.QueryContext(ctx, `select id, title, owner_id, status, created_at from courses order by created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.Course
	for rows.Next() {
		var item domain.Course
		if err := rows.Scan(&item.ID, &item.Title, &item.OwnerID, &item.Status, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) AddMember(ctx context.Context, member domain.CourseMember) error {
	_, err := r.db.ExecContext(ctx, `insert into course_members(course_id, user_id, role) values ($1,$2,$3) on conflict (course_id, user_id) do update set role=excluded.role`,
		member.CourseID, member.UserID, member.Role)
	return err
}

func (r *Repository) ListMembers(ctx context.Context, courseID string) ([]domain.CourseMember, error) {
	rows, err := r.db.QueryContext(ctx, `select course_id, user_id, role from course_members where course_id=$1 order by user_id`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.CourseMember
	for rows.Next() {
		var item domain.CourseMember
		if err := rows.Scan(&item.CourseID, &item.UserID, &item.Role); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) CreateAssignment(ctx context.Context, item domain.Assignment) error {
	_, err := r.db.ExecContext(ctx, `insert into assignments(id, course_id, title, work_id, task_ids, due_at, assigned_by, status, created_at) values ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		item.ID, item.CourseID, item.Title, item.WorkID, pg.Marshal(item.TaskIDs), nullableTime(item.DueAt), item.AssignedBy, item.Status, item.CreatedAt)
	return err
}

func (r *Repository) AssignmentExists(ctx context.Context, id string) (bool, error) {
	return exists(ctx, r.db, `select exists(select 1 from assignments where id=$1)`, id)
}

func (r *Repository) ListAssignments(ctx context.Context, courseID string) ([]domain.Assignment, error) {
	rows, err := r.db.QueryContext(ctx, `select id, course_id, title, work_id, task_ids, due_at, assigned_by, status, created_at from assignments where course_id=$1 order by created_at`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.Assignment
	for rows.Next() {
		item, err := scanAssignment(rows.Scan)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) CreateSubmission(ctx context.Context, item domain.Submission) error {
	_, err := r.db.ExecContext(ctx, `insert into submissions(id, assignment_id, user_id, answers_json, status, submitted_at, review_json) values ($1,$2,$3,$4,$5,$6,$7)`,
		item.ID, item.AssignmentID, item.UserID, pg.Marshal(item.Answers), item.Status, item.SubmittedAt, nil)
	return err
}

func (r *Repository) ListSubmissions(ctx context.Context, assignmentID string) ([]domain.Submission, error) {
	rows, err := r.db.QueryContext(ctx, `select id, assignment_id, user_id, answers_json, status, submitted_at, review_json from submissions where assignment_id=$1 order by submitted_at`, assignmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.Submission
	for rows.Next() {
		item, err := scanSubmission(rows.Scan)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) ReviewSubmission(ctx context.Context, id string, review domain.TeacherReview) (domain.Submission, bool, error) {
	_, err := r.db.ExecContext(ctx, `update submissions set review_json=$2, status='reviewed' where id=$1`, id, pg.Marshal(review))
	if err != nil {
		return domain.Submission{}, false, err
	}
	row := r.db.QueryRowContext(ctx, `select id, assignment_id, user_id, answers_json, status, submitted_at, review_json from submissions where id=$1`, id)
	item, err := scanSubmission(row.Scan)
	if err == sql.ErrNoRows {
		return domain.Submission{}, false, nil
	}
	return item, err == nil, err
}

func scanAssignment(scan func(dest ...any) error) (domain.Assignment, error) {
	var item domain.Assignment
	var taskRaw []byte
	var dueAt sql.NullTime
	if err := scan(&item.ID, &item.CourseID, &item.Title, &item.WorkID, &taskRaw, &dueAt, &item.AssignedBy, &item.Status, &item.CreatedAt); err != nil {
		return domain.Assignment{}, err
	}
	item.TaskIDs = pg.Unmarshal[[]string](taskRaw)
	if dueAt.Valid {
		item.DueAt = dueAt.Time
	}
	return item, nil
}

func scanSubmission(scan func(dest ...any) error) (domain.Submission, error) {
	var item domain.Submission
	var answersRaw, reviewRaw []byte
	if err := scan(&item.ID, &item.AssignmentID, &item.UserID, &answersRaw, &item.Status, &item.SubmittedAt, &reviewRaw); err != nil {
		return domain.Submission{}, err
	}
	item.Answers = pg.Unmarshal[[]domain.SubmissionAnswer](answersRaw)
	if len(reviewRaw) > 0 {
		review := pg.Unmarshal[domain.TeacherReview](reviewRaw)
		item.Review = &review
	}
	return item, nil
}

func nullableTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}

func exists(ctx context.Context, db *sql.DB, query, id string) (bool, error) {
	var value bool
	err := db.QueryRowContext(ctx, query, id).Scan(&value)
	return value, err
}
