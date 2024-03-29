package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"grpc-service-ref/internal/domain/models"
	"grpc-service-ref/internal/storage"

	"github.com/mattn/go-sqlite3"
)

type Storage struct {
	db *sql.DB
}

func New(storagePath string) (*Storage, error) {
	const op = "storage.sqlite.New"

	db, err := sql.Open("sqlite3", storagePath)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{db: db}, nil
}

func (s *Storage) Stop() error {
	return s.db.Close()
}

// SaveUser saves user to db.
func (s *Storage) SaveUser(ctx context.Context, email string, passHash []byte) (int64, error) {
	const op = "storage.sqlite.SaveUser"

	stmt, err := s.db.Prepare("INSERT INTO users(email, pass_hash) VALUES(?, ?)")
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	res, err := stmt.ExecContext(ctx, email, passHash)
	if err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
			return 0, fmt.Errorf("%s: %w", op, storage.ErrUserExists)
		}

		return 0, fmt.Errorf("%s: %w", op, err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return id, nil
}

func (s *Storage) UpdateUser(ctx context.Context, user models.User, passHash []byte) (int64, error) {
	const op = "storage.sqlite.updateuser"

	stmt, err := s.db.Prepare("UPDATE users SET email = ?, pass_hash = ?, is_verified = ? WHERE email = ?")
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	res, err := stmt.ExecContext(ctx, user.Email, passHash, user.Verified, user.Email)
	if err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == sql.ErrNoRows {
			return 0, fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
		}

		return 0, fmt.Errorf("%s: %w", op, err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return id, nil
}

func (s *Storage) VerifyUser(ctx context.Context, email string) (int64, error) {
	const op = "storage.sqlite.VerifyUser"

	stmt, err := s.db.Prepare("UPDATE users SET is_verified = true WHERE email = ?")
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	res, err := stmt.ExecContext(ctx, email)
	if err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == sql.ErrNoRows {
			return 0, fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
		}

		return 0, fmt.Errorf("%s: %w", op, err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return id, nil
}

// User returns user by email.
func (s *Storage) User(ctx context.Context, email string) (models.User, error) {
	const op = "storage.sqlite.User"

	stmt, err := s.db.Prepare("SELECT id, email, pass_hash FROM users WHERE email = ?")
	if err != nil {
		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}

	row := stmt.QueryRowContext(ctx, email)

	var user models.User
	err = row.Scan(&user.ID, &user.Email, &user.PassHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.User{}, fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
		}

		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}

	return user, nil
}

//func (s *Storage) SavePermission(ctx context.Context, userID int64, permission models.Permission, appID string) error {
//	const op = "storage.sqlite.SavePermission"
//
//	stmt, err := s.db.Prepare("INSERT INTO permissions(user_id, permission, app_id) VALUES(?, ?, ?)")
//	if err != nil {
//		return fmt.Errorf("%s: %w", op, err)
//	}
//
//	_, err = stmt.ExecContext(ctx, userID, permission, appID)
//	if err != nil {
//		return fmt.Errorf("%s: %w", op, err)
//	}
//
//	return nil
//}

// App returns app by id.
func (s *Storage) App(ctx context.Context, id int) (models.App, error) {
	const op = "storage.sqlite.App"

	stmt, err := s.db.Prepare("SELECT id, name, secret FROM apps WHERE id = ?")
	if err != nil {
		return models.App{}, fmt.Errorf("%s: %w", op, err)
	}

	row := stmt.QueryRowContext(ctx, id)

	var app models.App
	err = row.Scan(&app.ID, &app.Name, &app.Secret)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.App{}, fmt.Errorf("%s: %w", op, storage.ErrAppNotFound)
		}

		return models.App{}, fmt.Errorf("%s: %w", op, err)
	}

	return app, nil
}

func (s *Storage) IsAdmin(ctx context.Context, userID int64) (bool, error) {
	const op = "storage.sqlite.IsAdmin"

	stmt, err := s.db.Prepare("SELECT is_admin FROM users WHERE id = ?")
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	row := stmt.QueryRowContext(ctx, userID)

	var isAdmin bool

	err = row.Scan(&isAdmin)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
		}

		return false, fmt.Errorf("%s: %w", op, err)
	}

	return isAdmin, nil
}

func (s *Storage) StoreVerification(ctx context.Context, email string, code string, expiresAt time.Time) (models.VerificationData, error) {
	const op = "storage.sqlite.StoreVerification"

	stmt, err := s.db.Prepare("INSERT INTO verifications(email, code, expiresAt) VALUES(?, ?, ?)")
	if err != nil {
		return models.VerificationData{}, fmt.Errorf("%s: %w", op, err)
	}

	res, err := stmt.ExecContext(ctx, email, code, expiresAt)
	if err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
			return models.VerificationData{}, fmt.Errorf("%s: %w", op, storage.ErrUserExists)
		}

		return models.VerificationData{}, fmt.Errorf("%s: %w", op, err)
	}

	res.RowsAffected()
	// _, err := res.LastInsertId()
	// if err != nil {
	// 	return models.VerificationData{}, fmt.Errorf("%s: %w", op, err)
	// }

	return models.VerificationData{}, nil
}

func (s *Storage) Verification(ctx context.Context, email string) (models.VerificationData, error) {
	const op = "storage.sqlite.Verification"

	stmt, err := s.db.Prepare("SELECT * FROM verifications WHERE email = ?")
	if err != nil {
		return models.VerificationData{}, fmt.Errorf("%s: %w", op, err)
	}

	row := stmt.QueryRowContext(ctx, email)
	var verification models.VerificationData
	err = row.Scan(&verification.Email, &verification.Code, &verification.ExpiresAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.VerificationData{}, fmt.Errorf("%s: %w", op, storage.ErrVerificationNotFound)
		}
		return models.VerificationData{}, fmt.Errorf("%s: %w", op, err)
	}
	return verification, nil
}

func (s *Storage) DeleteVerification(ctx context.Context, email string) error {
	const op = "storage.sqlite.DeleteVerification"

	stmt, err := s.db.Prepare("DELETE from verifications WHERE email = ?")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	res, err := stmt.ExecContext(ctx, email)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	_ = res
	return nil
}
