package application

import (
	"context"
	"errors"
	"strings"
	"time"

	"itmo-lms/course-service/internal/domain"
	"itmo-lms/pkg/platform"
)

var ErrNotFound = errors.New("entity not found")

type Service struct {
	repo domain.Repository
}

func NewService(repo domain.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateCourse(ctx context.Context, course domain.Course) (domain.Course, error) {
	if strings.TrimSpace(course.Title) == "" || strings.TrimSpace(course.OwnerID) == "" {
		return domain.Course{}, errors.New("title and owner_id are required")
	}
	course.ID = platform.NewID("crs")
	course.Status = valueOr(course.Status, "active")
	course.CreatedAt = time.Now().UTC()
	return course, s.repo.CreateCourse(ctx, course)
}

func (s *Service) ListCourses(ctx context.Context) ([]domain.Course, error) {
	return s.repo.ListCourses(ctx)
}

func (s *Service) AddMember(ctx context.Context, courseID string, member domain.CourseMember) (domain.CourseMember, error) {
	ok, err := s.repo.CourseExists(ctx, courseID)
	if err != nil {
		return domain.CourseMember{}, err
	}
	if !ok {
		return domain.CourseMember{}, ErrNotFound
	}
	if member.UserID == "" {
		return domain.CourseMember{}, errors.New("user_id is required")
	}
	member.CourseID = courseID
	member.Role = valueOr(member.Role, "student")
	return member, s.repo.AddMember(ctx, member)
}

func (s *Service) ListMembers(ctx context.Context, courseID string) ([]domain.CourseMember, error) {
	return s.repo.ListMembers(ctx, courseID)
}

func (s *Service) CreateAssignment(ctx context.Context, courseID string, assignment domain.Assignment) (domain.Assignment, error) {
	ok, err := s.repo.CourseExists(ctx, courseID)
	if err != nil {
		return domain.Assignment{}, err
	}
	if !ok {
		return domain.Assignment{}, ErrNotFound
	}
	if assignment.Title == "" || (assignment.WorkID == "" && len(assignment.TaskIDs) == 0) {
		return domain.Assignment{}, errors.New("title and work_id or task_ids are required")
	}
	assignment.ID = platform.NewID("asg")
	assignment.CourseID = courseID
	assignment.Status = valueOr(assignment.Status, "published")
	assignment.CreatedAt = time.Now().UTC()
	return assignment, s.repo.CreateAssignment(ctx, assignment)
}

func (s *Service) ListAssignments(ctx context.Context, courseID string) ([]domain.Assignment, error) {
	return s.repo.ListAssignments(ctx, courseID)
}

func (s *Service) CreateSubmission(ctx context.Context, assignmentID string, submission domain.Submission) (domain.Submission, error) {
	ok, err := s.repo.AssignmentExists(ctx, assignmentID)
	if err != nil {
		return domain.Submission{}, err
	}
	if !ok {
		return domain.Submission{}, ErrNotFound
	}
	if submission.UserID == "" {
		return domain.Submission{}, errors.New("user_id is required")
	}
	submission.ID = platform.NewID("sub")
	submission.AssignmentID = assignmentID
	submission.Status = "submitted"
	submission.SubmittedAt = time.Now().UTC()
	return submission, s.repo.CreateSubmission(ctx, submission)
}

func (s *Service) ListSubmissions(ctx context.Context, assignmentID string) ([]domain.Submission, error) {
	return s.repo.ListSubmissions(ctx, assignmentID)
}

func (s *Service) ReviewSubmission(ctx context.Context, submissionID string, review domain.TeacherReview) (domain.Submission, error) {
	if review.ReviewerID == "" {
		return domain.Submission{}, errors.New("reviewer_id is required")
	}
	review.ReviewedAt = time.Now().UTC()
	submission, ok, err := s.repo.ReviewSubmission(ctx, submissionID, review)
	if err != nil {
		return domain.Submission{}, err
	}
	if !ok {
		return domain.Submission{}, ErrNotFound
	}
	return submission, nil
}

func valueOr(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
