package app

import (
	"log/slog"
	"time"

	grpcapp "grpc-service-ref/internal/app/grpc"
	"grpc-service-ref/internal/services/auth"
	"grpc-service-ref/internal/services/mail/gmail"
	"grpc-service-ref/internal/services/verification"
	"grpc-service-ref/internal/storage/sqlite"
)

type App struct {
	GRPCServer *grpcapp.App
}

func New(
	log *slog.Logger,
	grpcPort int,
	storagePath string,
	tokenTTL time.Duration,
	senderName string,
	senderEmail string,
	senderPassword string,
) *App {
	storage, err := sqlite.New(storagePath)
	if err != nil {
		panic(err)
	}

	authService := auth.New(log, storage, storage, storage, tokenTTL)
	mailService := gmail.New(log, senderName, senderEmail, senderPassword)
	verification := verification.New(log, storage, storage, storage, storage)
	grpcApp := grpcapp.New(log, authService, mailService, verification, grpcPort)

	return &App{
		GRPCServer: grpcApp,
	}
}
