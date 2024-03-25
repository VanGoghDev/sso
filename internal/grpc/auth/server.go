package authgrpc

import (
	"context"
	"errors"
	"time"

	"grpc-service-ref/internal/domain/models"
	"grpc-service-ref/internal/lib/verification"
	"grpc-service-ref/internal/services/auth"
	verificationService "grpc-service-ref/internal/services/verification"
	"grpc-service-ref/internal/storage"

	ssov1 "github.com/VanGoghDev/protos/gen/go/sso"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Authentication service
type Auth interface {
	Login(
		ctx context.Context,
		email string,
		password string,
		appID int,
	) (token string, err error)
	RegisterNewUser(
		ctx context.Context,
		email string,
		password string,
	) (userID int64, err error)
	IsAdmin(ctx context.Context, userID int64) (bool, error)
	UpdateUser(
		ctx context.Context,
		email string,
		password string,
	) (userID int64, err error)
}

type EmailSender interface {
	SendEmail(
		subject string,
		to []string,
		content string,
		cc []string,
		bcc []string,
		atachFiles []string,
	) error
}

// Verification service
type Verification interface {
	StoreVerification(
		ctx context.Context,
		email string,
		code string,
		expiresAt time.Time,
	) (verificationData models.VerificationData, err error)
	Verify(
		ctx context.Context,
		email string,
		code string,
		deleteVerificationAfterAtempt bool,
	) (result string, err error)
	DeleteVerification(
		ctx context.Context,
		email string,
	) error
}

type serverAPI struct {
	verificationCodeLen           int
	verificationExpiresAfterHours int
	ssov1.UnimplementedAuthServer
	auth         Auth
	verification Verification
	emailService EmailSender
}

func Register(gRPCServer *grpc.Server, auth Auth, emailService EmailSender, verification Verification, verificationCodeLen int, verificationExpiresAt int) {
	ssov1.RegisterAuthServer(gRPCServer, &serverAPI{auth: auth, emailService: emailService, verification: verification, verificationCodeLen: verificationCodeLen, verificationExpiresAfterHours: verificationExpiresAt})
}

func (s *serverAPI) Login(
	ctx context.Context,
	in *ssov1.LoginRequest,
) (*ssov1.LoginResponse, error) {
	if in.Email == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}

	if in.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "password is required")
	}

	if in.GetAppId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "app_id is required")
	}

	token, err := s.auth.Login(ctx, in.GetEmail(), in.GetPassword(), int(in.GetAppId()))
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			return nil, status.Error(codes.InvalidArgument, "invalid email or password")
		}

		return nil, status.Error(codes.Internal, "failed to login")
	}

	return &ssov1.LoginResponse{Token: token}, nil
}

func (s *serverAPI) Register(
	ctx context.Context,
	in *ssov1.RegisterRequest,
) (*ssov1.RegisterResponse, error) {
	if in.Email == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}

	if in.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "password is required")
	}

	// save user
	uid, err := s.auth.RegisterNewUser(ctx, in.GetEmail(), in.GetPassword())
	if err != nil {
		if errors.Is(err, storage.ErrUserExists) {
			return nil, status.Error(codes.AlreadyExists, "user already exists")
		}

		return nil, status.Error(codes.Internal, "failed to register user")
	}
	verificationCode := verification.GenerateRandomString(s.verificationCodeLen)
	// save verification data
	result, err := s.verification.StoreVerification(ctx, in.GetEmail(), verificationCode, time.Now().UTC().Add(time.Hour*time.Duration(s.verificationExpiresAfterHours)))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to register user")
	}

	// send verification email
	if err := s.emailService.SendEmail("Verify your new account", []string{in.GetEmail()}, verificationCode, []string{}, []string{}, []string{}); err != nil {
		return nil, status.Error(codes.Internal, "failed to send email")
	}
	_ = result

	return &ssov1.RegisterResponse{UserId: uid}, nil
}

func (s *serverAPI) IsAdmin(
	ctx context.Context,
	in *ssov1.IsAdminRequest,
) (*ssov1.IsAdminResponse, error) {
	if in.GetUserId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	isAdmin, err := s.auth.IsAdmin(ctx, in.GetUserId())
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}

		return nil, status.Error(codes.Internal, "failed to check admin status")
	}

	return &ssov1.IsAdminResponse{IsAdmin: isAdmin}, nil
}

func (s *serverAPI) CreateVerification(
	ctx context.Context,
	in *ssov1.CreateVerificationRequest,
) (*ssov1.CreateVerificationResponse, error) {
	if in.GetEmail() == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}

	verificationCode := verification.GenerateRandomString(s.verificationCodeLen)
	// save verification data
	result, err := s.verification.StoreVerification(ctx, in.GetEmail(), verificationCode, time.Now().UTC().Add(time.Hour*time.Duration(s.verificationExpiresAfterHours)))
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, "unable to create verification with email provided")
		}

		return nil, status.Error(codes.Internal, "failed to create verification")
	}
	_ = result

	// send code to email
	if err := s.emailService.SendEmail("Verify your new account", []string{in.GetEmail()}, verificationCode, []string{}, []string{}, []string{}); err != nil {
		return nil, status.Error(codes.Internal, "failed to send email")
	}

	return &ssov1.CreateVerificationResponse{Success: true}, nil
}

func (s *serverAPI) VerifyMail(
	ctx context.Context,
	in *ssov1.VerifyMailRequest,
) (*ssov1.VerifyMailResponse, error) {
	if in.GetEmail() == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}

	if in.GetCode() == "" {
		return nil, status.Error(codes.InvalidArgument, "code is required")
	}

	result, err := s.verification.Verify(ctx, in.GetEmail(), in.GetCode(), true)
	if success, err := validateVerificationResult(err); !success {
		return nil, err
	}

	return &ssov1.VerifyMailResponse{Result: result}, nil
}

func (s *serverAPI) ResetPassword(
	ctx context.Context,
	in *ssov1.ResetPasswordRequest,
) (*ssov1.ResetPasswordResponse, error) {
	if in.GetEmail() == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}

	if in.GetCode() == "" {
		return nil, status.Error(codes.InvalidArgument, "code is required")
	}

	if in.GetNewPassword() == "" {
		return nil, status.Error(codes.InvalidArgument, "password is required")
	}

	verificationResult, err := s.verification.Verify(ctx, in.GetEmail(), in.GetCode(), false)

	if success, err := validateVerificationResult(err); !success {
		return nil, err
	}

	uid, err := s.auth.UpdateUser(ctx, in.GetEmail(), in.GetNewPassword())
	if err != nil {
		if errors.Is(err, auth.ErrPassAreEqual) {
			return nil, status.Error(codes.InvalidArgument, "passwords should differ")
		}

		return nil, status.Error(codes.Internal, "failed to update user password")
	}

	_ = uid
	_ = verificationResult

	err = s.verification.DeleteVerification(ctx, in.GetEmail())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to delete verification")
	}

	return &ssov1.ResetPasswordResponse{Success: true}, nil
}

func validateVerificationResult(err error) (bool, error) {
	if err != nil {
		if errors.Is(err, storage.ErrVerificationNotFound) {
			return false, status.Error(codes.NotFound, "verification not found")
		}
		if errors.Is(err, storage.ErrVerificationExpired) {
			return false, status.Error(codes.Internal, "verification expired")
		}
		if errors.Is(err, verificationService.CodesDiffer) {
			return false, status.Error(codes.PermissionDenied, "codes differ")
		}

		return false, status.Error(codes.Internal, "failed to verify email")
	}

	return true, nil
}
