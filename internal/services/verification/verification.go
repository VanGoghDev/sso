package verification

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"grpc-service-ref/internal/domain/models"
	"grpc-service-ref/internal/lib/logger/sl"
	"grpc-service-ref/internal/services/auth"
	"grpc-service-ref/internal/storage"
)

type VerificationSaver interface {
	StoreVerification(
		ctx context.Context,
		email string,
		code string,
		expiresAt time.Time,
	) (verificationData models.VerificationData, err error)
}

type VerificationProvider interface {
	Verification(ctx context.Context, email string) (verificationData models.VerificationData, err error)
}

type VerificationDeleter interface {
	DeleteVerification(ctx context.Context, email string) error
}

type Verification struct {
	log                  *slog.Logger
	verificationSaver    VerificationSaver
	verificationProvider VerificationProvider
	verificationDeleter  VerificationDeleter
	userSaver            auth.UserSaver
}

var (
	EmptyEmail          = errors.New("Empty email")
	EmptyCode           = errors.New("Empty code")
	EmptyExpiresAt      = errors.New("Empty expires at")
	CodesDiffer         = errors.New("Codes are different")
	VerificationExpired = errors.New("Verification expired")
)

func New(
	log *slog.Logger,
	verificationSaver VerificationSaver,
	verificationProvider VerificationProvider,
	verificationDeleter VerificationDeleter,
	userSaver auth.UserSaver,
) *Verification {
	return &Verification{
		log:                  log,
		verificationSaver:    verificationSaver,
		verificationProvider: verificationProvider,
		verificationDeleter:  verificationDeleter,
		userSaver:            userSaver,
	}
}

func (v *Verification) StoreVerification(
	ctx context.Context,
	email string,
	code string,
	expiresAt time.Time,
) (models.VerificationData, error) {
	const op = "Verification.StoreVerification"

	log := v.log.With(
		slog.String("op", op),
		slog.String("username", email),
	)

	log.Info("storing verification")

	if email == "" {
		log.Error("empty email")

		return models.VerificationData{}, fmt.Errorf("%s: %w", op, EmptyEmail)
	}

	if code == "" {
		log.Error("empty code")
		return models.VerificationData{}, fmt.Errorf("%s: %w", op, EmptyCode)
	}

	if time.Time.IsZero(expiresAt) {
		log.Error("empty expiresAt")

		return models.VerificationData{}, fmt.Errorf("%s: %w", op, EmptyExpiresAt)
	}

	verificationData, err := v.verificationSaver.StoreVerification(ctx, email, code, expiresAt)

	if err != nil {
		log.Error("failed to save verification data", sl.Err(err))

		return models.VerificationData{}, fmt.Errorf("%s: %w", op, err)
	}

	return verificationData, nil
}

func (v *Verification) Verify(
	ctx context.Context,
	email string,
	code string,
	deleteVerificationAfterAtempt bool,
) (string, error) {
	const op = "Verification.Verify"

	log := v.log.With(
		slog.String("op", op),
		slog.String("username", email),
	)

	if email == "" {
		log.Error("empty email")

		return "", fmt.Errorf("%s: %w", op, EmptyEmail)
	}

	if code == "" {
		log.Error("empty code")
		return "", fmt.Errorf("%s: %w", op, EmptyCode)
	}

	verification, err := v.verificationProvider.Verification(ctx, email)
	if err != nil {
		log.Error("failed to fetch verification data", sl.Err(err))
	}

	if verification.Code != code {
		return "", fmt.Errorf("%s, %w", op, CodesDiffer)
	}

	if verification.ExpiresAt.Before(time.Now()) {
		v.verificationDeleter.DeleteVerification(ctx, email)
		return "", fmt.Errorf("%s: %w", op, storage.ErrVerificationExpired)
	}

	// обновить юзера
	id, err := v.userSaver.VerifyUser(ctx, email)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	// удалить верификацию
	if deleteVerificationAfterAtempt {
		if err := v.verificationDeleter.DeleteVerification(ctx, email); err != nil {
			return "", fmt.Errorf("%s: %w", op, err)
		}
	}

	return fmt.Sprintf("%v", id), nil
}

func (v *Verification) DeleteVerification(
	ctx context.Context,
	email string,
) error {
	const op = "Verification.Delete"

	log := v.log.With(
		slog.String("op", op),
		slog.String("username", email),
	)
	log.Info("Deleting verification")

	if email == "" {
		log.Error("empty email")

		return fmt.Errorf("%s: %w", op, EmptyEmail)
	}

	if err := v.verificationDeleter.DeleteVerification(ctx, email); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}
