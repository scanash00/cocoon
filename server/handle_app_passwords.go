package server

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/haileyok/cocoon/internal/helpers"
	"github.com/haileyok/cocoon/models"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

type ComAtprotoServerCreateAppPasswordRequest struct {
	Name       string `json:"name" validate:"required"`
	Privileged *bool  `json:"privileged,omitempty"`
}

type ComAtprotoServerCreateAppPasswordResponse struct {
	Name       string `json:"name"`
	Password   string `json:"password"`
	CreatedAt  string `json:"createdAt"`
	Privileged bool   `json:"privileged"`
}

func generateAppPassword() string {
	b := make([]byte, 8)
	rand.Read(b)
	h := hex.EncodeToString(b)
	return h[0:4] + "-" + h[4:8] + "-" + h[8:12] + "-" + h[12:16]
}

func (s *Server) handleCreateAppPassword(e echo.Context) error {
	ctx := e.Request().Context()

	repo, ok := getRepoFromContext(e)
	if !ok {
		return helpers.UnauthorizedError(e, to.StringPtr("AuthMissing"))
	}

	var req ComAtprotoServerCreateAppPasswordRequest
	if err := e.Bind(&req); err != nil {
		return helpers.InputError(e, to.StringPtr("InvalidRequest"))
	}

	if err := e.Validate(req); err != nil {
		return helpers.InputError(e, to.StringPtr("InvalidRequest"))
	}

	var existing models.AppPassword
	if err := s.db.First(ctx, &existing, "did = ? AND name = ?", repo.Repo.Did, req.Name).Error; err == nil {
		return helpers.InputError(e, to.StringPtr("DuplicateName"))
	}

	password := generateAppPassword()
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("error hashing app password", "error", err)
		return helpers.ServerError(e, nil)
	}

	privileged := false
	if req.Privileged != nil {
		privileged = *req.Privileged
	}

	now := time.Now()
	appPassword := models.AppPassword{
		Name:       req.Name,
		Did:        repo.Repo.Did,
		Password:   string(hashedPassword),
		Privileged: privileged,
		CreatedAt:  now,
	}

	if err := s.db.Create(ctx, &appPassword, nil).Error; err != nil {
		s.logger.Error("error creating app password", "error", err)
		return helpers.ServerError(e, nil)
	}

	return e.JSON(200, ComAtprotoServerCreateAppPasswordResponse{
		Name:       req.Name,
		Password:   password,
		CreatedAt:  now.UTC().Format(time.RFC3339),
		Privileged: privileged,
	})
}

type AppPasswordInfo struct {
	Name       string `json:"name"`
	CreatedAt  string `json:"createdAt"`
	Privileged bool   `json:"privileged"`
}

type ComAtprotoServerListAppPasswordsResponse struct {
	Passwords []AppPasswordInfo `json:"passwords"`
}

func (s *Server) handleListAppPasswords(e echo.Context) error {
	ctx := e.Request().Context()

	repo, ok := getRepoFromContext(e)
	if !ok {
		return helpers.UnauthorizedError(e, to.StringPtr("AuthMissing"))
	}

	var appPasswords []models.AppPassword
	if err := s.db.Raw(ctx, "SELECT * FROM app_passwords WHERE did = ?", nil, repo.Repo.Did).Scan(&appPasswords).Error; err != nil {
		s.logger.Error("error listing app passwords", "error", err)
		return helpers.ServerError(e, nil)
	}

	passwords := make([]AppPasswordInfo, 0, len(appPasswords))
	for _, ap := range appPasswords {
		passwords = append(passwords, AppPasswordInfo{
			Name:       ap.Name,
			CreatedAt:  ap.CreatedAt.UTC().Format(time.RFC3339),
			Privileged: ap.Privileged,
		})
	}

	return e.JSON(200, ComAtprotoServerListAppPasswordsResponse{
		Passwords: passwords,
	})
}

type ComAtprotoServerRevokeAppPasswordRequest struct {
	Name string `json:"name" validate:"required"`
}

func (s *Server) handleRevokeAppPassword(e echo.Context) error {
	ctx := e.Request().Context()

	repo, ok := getRepoFromContext(e)
	if !ok {
		return helpers.UnauthorizedError(e, to.StringPtr("AuthMissing"))
	}

	var req ComAtprotoServerRevokeAppPasswordRequest
	if err := e.Bind(&req); err != nil {
		return helpers.InputError(e, to.StringPtr("InvalidRequest"))
	}

	if err := e.Validate(req); err != nil {
		return helpers.InputError(e, to.StringPtr("InvalidRequest"))
	}

	result := s.db.Exec(ctx, "DELETE FROM app_passwords WHERE did = ? AND name = ?", nil, repo.Repo.Did, req.Name)
	if result.Error != nil {
		s.logger.Error("error revoking app password", "error", result.Error)
		return helpers.ServerError(e, nil)
	}

	if result.RowsAffected == 0 {
		return helpers.InputError(e, to.StringPtr("AppPasswordNotFound"))
	}

	return e.JSON(200, map[string]any{})
}
