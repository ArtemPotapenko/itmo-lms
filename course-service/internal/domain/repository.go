package domain

import "context"

type Repository interface {
	CreateCourse(context.Context, Course) error
	CourseExists(context.Context, string) (bool, error)
	ListCourses(context.Context) ([]Course, error)
	AddMember(context.Context, CourseMember) error
	ListMembers(context.Context, string) ([]CourseMember, error)
	CreateAssignment(context.Context, Assignment) error
	AssignmentExists(context.Context, string) (bool, error)
	ListAssignments(context.Context, string) ([]Assignment, error)
	CreateSubmission(context.Context, Submission) error
	ListSubmissions(context.Context, string) ([]Submission, error)
	ReviewSubmission(context.Context, string, TeacherReview) (Submission, bool, error)
}
