package grpctransport

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	coursev1 "itmo-lms/course-service/gen"
	"itmo-lms/course-service/internal/application"
	"itmo-lms/course-service/internal/domain"
)

type Server struct {
	coursev1.UnimplementedCourseServiceServer
	service *application.Service
}

func New(service *application.Service) *Server { return &Server{service: service} }

func (s *Server) ListAssignments(ctx context.Context, req *coursev1.ListAssignmentsRequest) (*coursev1.ListAssignmentsReply, error) {
	items, err := s.service.ListAssignments(ctx, req.GetCourseId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	resp := &coursev1.ListAssignmentsReply{Items: make([]*coursev1.AssignmentReply, 0, len(items))}
	for _, item := range items {
		reply := &coursev1.AssignmentReply{
			Id:         item.ID,
			CourseId:   item.CourseID,
			Title:      item.Title,
			WorkId:     item.WorkID,
			TaskIds:    item.TaskIDs,
			AssignedBy: item.AssignedBy,
			Status:     item.Status,
			CreatedAt:  item.CreatedAt.Format(time.RFC3339),
		}
		if !item.DueAt.IsZero() {
			reply.DueAt = item.DueAt.Format(time.RFC3339)
		}
		resp.Items = append(resp.Items, reply)
	}
	return resp, nil
}

func (s *Server) ListMembers(ctx context.Context, req *coursev1.ListMembersRequest) (*coursev1.ListMembersReply, error) {
	items, err := s.service.ListMembers(ctx, req.GetCourseId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	resp := &coursev1.ListMembersReply{Items: make([]*coursev1.CourseMemberReply, 0, len(items))}
	for _, item := range items {
		resp.Items = append(resp.Items, &coursev1.CourseMemberReply{CourseId: item.CourseID, UserId: item.UserID, Role: item.Role})
	}
	return resp, nil
}

func (s *Server) CreateSubmission(ctx context.Context, req *coursev1.CreateSubmissionRequest) (*coursev1.SubmissionReply, error) {
	answers := make([]domain.SubmissionAnswer, 0, len(req.GetAnswers()))
	for _, item := range req.GetAnswers() {
		var correct *bool
		if item.GetHasIsCorrect() {
			value := item.GetIsCorrect()
			correct = &value
		}
		answers = append(answers, domain.SubmissionAnswer{ContentID: item.GetContentId(), Answer: item.GetAnswer(), IsCorrect: correct})
	}
	submission, err := s.service.CreateSubmission(ctx, req.GetAssignmentId(), domain.Submission{UserID: req.GetUserId(), Answers: answers})
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return submissionReply(submission), nil
}

func submissionReply(item domain.Submission) *coursev1.SubmissionReply {
	reply := &coursev1.SubmissionReply{
		Id:           item.ID,
		AssignmentId: item.AssignmentID,
		UserId:       item.UserID,
		Status:       item.Status,
		SubmittedAt:  item.SubmittedAt.Format(time.RFC3339),
		Answers:      make([]*coursev1.SubmissionAnswer, 0, len(item.Answers)),
	}
	for _, answer := range item.Answers {
		entry := &coursev1.SubmissionAnswer{ContentId: answer.ContentID, Answer: answer.Answer}
		if answer.IsCorrect != nil {
			entry.HasIsCorrect = true
			entry.IsCorrect = *answer.IsCorrect
		}
		reply.Answers = append(reply.Answers, entry)
	}
	return reply
}
