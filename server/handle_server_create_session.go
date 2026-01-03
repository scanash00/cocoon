package server

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/haileyok/cocoon/internal/helpers"
	"github.com/haileyok/cocoon/models"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type ComAtprotoServerCreateSessionRequest struct {
	Identifier      string  `json:"identifier" validate:"required"`
	Password        string  `json:"password" validate:"required"`
	AuthFactorToken *string `json:"authFactorToken,omitempty"`
}

type ComAtprotoServerCreateSessionResponse struct {
	AccessJwt       string  `json:"accessJwt"`
	RefreshJwt      string  `json:"refreshJwt"`
	Handle          string  `json:"handle"`
	Did             string  `json:"did"`
	Email           string  `json:"email"`
	EmailConfirmed  bool    `json:"emailConfirmed"`
	EmailAuthFactor bool    `json:"emailAuthFactor"`
	Active          bool    `json:"active"`
	Status          *string `json:"status,omitempty"`
}

func (s *Server) handleCreateSession(e echo.Context) error {
	ctx := e.Request().Context()

	var req ComAtprotoServerCreateSessionRequest
	if err := e.Bind(&req); err != nil {
		s.logger.Error("error binding request", "endpoint", "com.atproto.server.serverCreateSession", "error", err)
		return helpers.ServerError(e, nil)
	}

	if err := e.Validate(req); err != nil {
		var verr ValidationError
		if errors.As(err, &verr) {
			if verr.Field == "Identifier" {
				return helpers.InputError(e, to.StringPtr("InvalidRequest"))
			}

			if verr.Field == "Password" {
				return helpers.InputError(e, to.StringPtr("InvalidRequest"))
			}
		}
	}

	req.Identifier = strings.ToLower(req.Identifier)
	var idtype string
	if _, err := syntax.ParseDID(req.Identifier); err == nil {
		idtype = "did"
	} else if _, err := syntax.ParseHandle(req.Identifier); err == nil {
		idtype = "handle"
	} else {
		idtype = "email"
	}

	var repo models.RepoActor
	var err error
	switch idtype {
	case "did":
		err = s.db.Raw(ctx, "SELECT r.*, a.* FROM repos r LEFT JOIN actors a ON r.did = a.did WHERE r.did = ?", nil, req.Identifier).Scan(&repo).Error
	case "handle":
		err = s.db.Raw(ctx, "SELECT r.*, a.* FROM actors a LEFT JOIN repos r ON a.did = r.did WHERE a.handle = ?", nil, req.Identifier).Scan(&repo).Error
	case "email":
		err = s.db.Raw(ctx, "SELECT r.*, a.* FROM repos r LEFT JOIN actors a ON r.did = a.did WHERE r.email = ?", nil, req.Identifier).Scan(&repo).Error
	}

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return helpers.InputError(e, to.StringPtr("InvalidRequest"))
		}

		s.logger.Error("erorr looking up repo", "endpoint", "com.atproto.server.createSession", "error", err)
		return helpers.ServerError(e, nil)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(repo.Password), []byte(req.Password)); err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			var appPasswords []models.AppPassword
			if dbErr := s.db.Raw(ctx, "SELECT * FROM app_passwords WHERE did = ?", nil, repo.Repo.Did).Scan(&appPasswords).Error; dbErr == nil {
				for _, ap := range appPasswords {
					if bcrypt.CompareHashAndPassword([]byte(ap.Password), []byte(req.Password)) == nil {
						goto passwordValid
					}
				}
			}
		} else {
			s.logger.Error("error comparing hash and password", "error", err)
		}
		return helpers.InputError(e, to.StringPtr("InvalidRequest"))
	}
passwordValid:

	if repo.EmailAuthFactorEnabled {
		if req.AuthFactorToken == nil || *req.AuthFactorToken == "" {
			code := fmt.Sprintf("%06d", rand.Intn(1000000))
			expiresAt := time.Now().Add(10 * time.Minute)

			if err := s.db.Exec(ctx, "UPDATE repos SET email_auth_factor_code = ?, email_auth_factor_code_expires_at = ? WHERE did = ?",
				nil, code, expiresAt, repo.Repo.Did).Error; err != nil {
				s.logger.Error("error storing email auth factor code", "error", err)
				return helpers.ServerError(e, nil)
			}

			if s.mail != nil {
				s.mailLk.Lock()
				s.mail.To(repo.Email)
				s.mail.Subject("Your sign-in code")
				s.mail.HTML().Set(fmt.Sprintf(`
					<h1>Sign-in code</h1>
					<p>Your sign-in code is: <strong>%s</strong></p>
					<p>This code will expire in 10 minutes.</p>
				`, code))
				if err := s.mail.Send(); err != nil {
					s.mailLk.Unlock()
					s.logger.Error("error sending email auth factor code", "error", err)
					return helpers.ServerError(e, nil)
				}
				s.mailLk.Unlock()
			}

			return e.JSON(401, map[string]string{
				"error":   "AuthFactorTokenRequired",
				"message": "A sign-in code has been sent to your email address",
			})
		}

		if repo.EmailAuthFactorCode == nil || *repo.EmailAuthFactorCode != *req.AuthFactorToken {
			return helpers.InputError(e, to.StringPtr("InvalidToken"))
		}

		if repo.EmailAuthFactorCodeExpiresAt == nil || time.Now().After(*repo.EmailAuthFactorCodeExpiresAt) {
			return helpers.InputError(e, to.StringPtr("ExpiredToken"))
		}

		if err := s.db.Exec(ctx, "UPDATE repos SET email_auth_factor_code = NULL, email_auth_factor_code_expires_at = NULL WHERE did = ?",
			nil, repo.Repo.Did).Error; err != nil {
			s.logger.Error("error clearing email auth factor code", "error", err)
		}
	}

	sess, err := s.createSession(ctx, &repo.Repo)
	if err != nil {
		s.logger.Error("error creating session", "error", err)
		return helpers.ServerError(e, nil)
	}

	return e.JSON(200, ComAtprotoServerCreateSessionResponse{
		AccessJwt:       sess.AccessToken,
		RefreshJwt:      sess.RefreshToken,
		Handle:          repo.Handle,
		Did:             repo.Repo.Did,
		Email:           repo.Email,
		EmailConfirmed:  repo.EmailConfirmedAt != nil,
		EmailAuthFactor: repo.EmailAuthFactorEnabled,
		Active:          repo.Active(),
		Status:          repo.Status(),
	})
}
