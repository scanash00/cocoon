package server

import (

	"fmt"
	"io"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/haileyok/cocoon/internal/helpers"
	"github.com/haileyok/cocoon/models"
	"github.com/ipfs/go-cid"
	"github.com/labstack/echo/v4"
)

func (s *Server) handleSyncGetBlob(e echo.Context) error {
	ctx := e.Request().Context()

	did := e.QueryParam("did")
	if did == "" {
		return helpers.InputError(e, to.StringPtr("InvalidRequest"))
	}

	cstr := e.QueryParam("cid")
	if cstr == "" {
		return helpers.InputError(e, to.StringPtr("InvalidRequest"))
	}

	c, err := cid.Parse(cstr)
	if err != nil {
		return helpers.InputError(e, to.StringPtr("InvalidRequest"))
	}

	urepo, err := s.getRepoActorByDid(ctx, did)
	if err != nil {
		s.logger.Error("could not find user for requested blob", "error", err)
		return helpers.InputError(e, to.StringPtr("InvalidRequest"))
	}

	status := urepo.Status()
	if status != nil {
		if *status == "deactivated" {
			return helpers.InputError(e, to.StringPtr("RepoDeactivated"))
		}
	}

	var blob models.Blob
	if err := s.db.Raw(ctx, "SELECT * FROM blobs WHERE did = ? AND cid = ?", nil, did, c.Bytes()).Scan(&blob).Error; err != nil {
		s.logger.Error("error looking up blob", "error", err)
		return helpers.ServerError(e, nil)
	}

	if blob.ID == 0 {
		s.logger.Error("blob not found", "did", did, "cid", cstr)
		return helpers.InputError(e, to.StringPtr("BlobNotFound"))
	}

	if blob.Storage == "s3" {
		if !(s.s3Config != nil && s.s3Config.BlobstoreEnabled) {
			s.logger.Error("s3 storage disabled but blob references s3")
			return helpers.ServerError(e, nil)
		}

		blobKey := fmt.Sprintf("blobs/%s/%s", urepo.Repo.Did, c.String())

		if s.s3Config.CDNUrl != "" {
			redirectUrl := fmt.Sprintf("%s/%s", s.s3Config.CDNUrl, blobKey)
			return e.Redirect(302, redirectUrl)
		}

		config := &aws.Config{
			Region:      aws.String(s.s3Config.Region),
			Credentials: credentials.NewStaticCredentials(s.s3Config.AccessKey, s.s3Config.SecretKey, ""),
		}

		if s.s3Config.Endpoint != "" {
			config.Endpoint = aws.String(s.s3Config.Endpoint)
			config.S3ForcePathStyle = aws.Bool(true)
		}

		sess, err := session.NewSession(config)
		if err != nil {
			s.logger.Error("error creating aws session", "error", err)
			return helpers.ServerError(e, nil)
		}

		svc := s3.New(sess)
		result, err := svc.GetObject(&s3.GetObjectInput{
			Bucket: aws.String(s.s3Config.Bucket),
			Key:    aws.String(blobKey),
		})
		if err != nil {
			s.logger.Error("error getting blob from s3", "error", err)
			return helpers.ServerError(e, nil)
		}
		defer result.Body.Close()

		e.Response().Header().Set(echo.HeaderContentDisposition, "attachment; filename="+c.String())
		e.Response().Header().Set(echo.HeaderContentType, "application/octet-stream")
		e.Response().WriteHeader(200)

		if _, err := io.Copy(e.Response().Writer, result.Body); err != nil {
			s.logger.Error("error streaming blob from s3", "error", err)
			return nil
		}
		return nil
	}

	if blob.Storage == "sqlite" {
		rows, err := s.db.Raw(ctx, "SELECT data FROM blob_parts WHERE blob_id = ? ORDER BY idx", nil, blob.ID).Rows()
		if err != nil {
			s.logger.Error("error getting blob parts", "error", err)
			return helpers.ServerError(e, nil)
		}
		defer rows.Close()

		e.Response().Header().Set(echo.HeaderContentDisposition, "attachment; filename="+c.String())
		e.Response().Header().Set(echo.HeaderContentType, "application/octet-stream")
		e.Response().WriteHeader(200)

		var data []byte
		for rows.Next() {
			if err := rows.Scan(&data); err != nil {
				s.logger.Error("error scanning blob part", "error", err)
				continue
			}
			if _, err := e.Response().Write(data); err != nil {
				return nil
			}
		}
		return nil
	}

	s.logger.Error("unknown storage type", "storage", blob.Storage)
	return helpers.ServerError(e, nil)
}
