package transport

import (
	"context"
	"errors"
	"log"
	"net/mail"
	"strings"
	"unicode/utf8"

	"authservice/internal/models"
	"authservice/internal/repository"
	"authservice/internal/service"
	"authservice/internal/token"
	authpb "authservice/proto/auth"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Users is implemented by the user service from part 1.
type Users interface {
	Register(context.Context, string, string, string, string, string) (*models.User, error)
	Login(context.Context, string, string) (*models.User, error)
	ChangePassword(context.Context, string, string, string) error
	ResetPassword(context.Context, string) error
}

type Server struct {
	authpb.UnimplementedAuthServiceServer
	users  Users
	tokens token.Manager
}

func New(users Users, tokens token.Manager) *Server {
	return &Server{users: users, tokens: tokens}
}

func (s *Server) Register(ctx context.Context, req *authpb.RegisterRequest) (*authpb.AuthResponse, error) {
	if !validEmail(req.GetEmail()) || !validText(req.GetFirstName(), 100) ||
		!validText(req.GetLastName(), 100) || !validText(req.GetPhone(), 20) || !validPassword(req.GetPassword()) {
		return nil, status.Error(codes.InvalidArgument, "email, names, phone and a password of 1-72 bytes are required")
	}
	user, err := s.users.Register(ctx, req.Email, req.FirstName, req.LastName, req.Phone, req.Password)
	if err != nil {
		return nil, rpcError(err)
	}
	return s.issue(user)
}

func (s *Server) Login(ctx context.Context, req *authpb.LoginRequest) (*authpb.AuthResponse, error) {
	if !validEmail(req.GetEmail()) || !validPassword(req.GetPassword()) {
		return nil, status.Error(codes.InvalidArgument, "valid email and password are required")
	}
	user, err := s.users.Login(ctx, req.Email, req.Password)
	if err != nil {
		return nil, rpcError(err)
	}
	return s.issue(user)
}

func (s *Server) ResetPassword(ctx context.Context, req *authpb.ResetPasswordRequest) (*authpb.ResetPasswordResponse, error) {
	if !validEmail(req.GetEmail()) {
		return nil, status.Error(codes.InvalidArgument, "valid email is required")
	}
	// The public reset endpoint must not reveal which emails are registered.
	if err := s.users.ResetPassword(ctx, req.Email); err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, rpcError(err)
	}
	return &authpb.ResetPasswordResponse{Success: true}, nil
}

func (s *Server) ValidateToken(_ context.Context, req *authpb.ValidateTokenRequest) (*authpb.ValidateTokenResponse, error) {
	claims, err := s.tokens.Validate(req.GetToken())
	if err != nil {
		return &authpb.ValidateTokenResponse{IsValid: false}, nil
	}
	return &authpb.ValidateTokenResponse{IsValid: true, UserId: claims.Subject, Email: claims.Email}, nil
}

func (s *Server) ChangePassword(ctx context.Context, req *authpb.ChangePasswordRequest) (*authpb.ChangePasswordResponse, error) {
	claims, err := s.tokens.Validate(req.GetToken())
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "valid token is required")
	}
	if !validPassword(req.GetOldPassword()) || !validPassword(req.GetNewPassword()) {
		return nil, status.Error(codes.InvalidArgument, "passwords must contain 1-72 bytes")
	}
	if err := s.users.ChangePassword(ctx, claims.Subject, req.OldPassword, req.NewPassword); err != nil {
		return nil, rpcError(err)
	}
	return &authpb.ChangePasswordResponse{Success: true}, nil
}

func (s *Server) issue(user *models.User) (*authpb.AuthResponse, error) {
	raw, err := s.tokens.Issue(user.ID.String(), user.Email)
	if err != nil {
		return nil, rpcError(err)
	}
	return &authpb.AuthResponse{Token: raw}, nil
}

func rpcError(err error) error {
	switch {
	case errors.Is(err, service.ErrEmailExists):
		return status.Error(codes.AlreadyExists, "email is already registered")
	case errors.Is(err, service.ErrInvalidCredentials):
		return status.Error(codes.Unauthenticated, "invalid credentials")
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return status.FromContextError(err).Err()
	default:
		log.Printf("auth operation failed: %v", err)
		return status.Error(codes.Internal, "authentication service error")
	}
}

func validEmail(value string) bool {
	address, err := mail.ParseAddress(value)
	return err == nil && address.Address == value && utf8.RuneCountInString(value) <= 100
}

func validText(value string, max int) bool {
	return strings.TrimSpace(value) != "" && !strings.ContainsRune(value, 0) && utf8.RuneCountInString(value) <= max
}

func validPassword(value string) bool {
	// bcrypt rejects inputs longer than 72 bytes, including multibyte characters.
	return len(value) > 0 && len(value) <= 72
}
