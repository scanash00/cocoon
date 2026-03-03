package server

import (
	"context"
	"net/url"

	"github.com/bluesky-social/indigo/atproto/atdata"
	"github.com/haileyok/cocoon/models"
)

// getAvatarURL returns a URL to retrieve a user's avatar blob, or "" if none is set.
func (s *Server) getAvatarURL(ctx context.Context, did string) string {
	var record models.Record
	if err := s.db.Raw(ctx, "SELECT * FROM records WHERE did = ? AND nsid = ? AND rkey = ?", nil, did, "app.bsky.actor.profile", "self").Scan(&record).Error; err != nil || record.Did == "" {
		return ""
	}
	val, err := atdata.UnmarshalCBOR(record.Value)
	if err != nil {
		return ""
	}
	avatarRaw, ok := val["avatar"]
	if !ok {
		return ""
	}
	blob, ok := avatarRaw.(atdata.Blob)
	if !ok {
		return ""
	}
	cidStr := blob.Ref.String()
	if cidStr == "" {
		return ""
	}
	return "/xrpc/com.atproto.sync.getBlob?did=" + url.QueryEscape(did) + "&cid=" + url.QueryEscape(cidStr)
}

func (s *Server) getActorByHandle(ctx context.Context, handle string) (*models.Actor, error) {
	var actor models.Actor
	if err := s.db.First(ctx, &actor, models.Actor{Handle: handle}).Error; err != nil {
		return nil, err
	}
	return &actor, nil
}

func (s *Server) getRepoByEmail(ctx context.Context, email string) (*models.Repo, error) {
	var repo models.Repo
	if err := s.db.First(ctx, &repo, models.Repo{Email: email}).Error; err != nil {
		return nil, err
	}
	return &repo, nil
}

func (s *Server) getRepoActorByEmail(ctx context.Context, email string) (*models.RepoActor, error) {
	var repo models.RepoActor
	if err := s.db.Raw(ctx, "SELECT r.*, a.* FROM repos r LEFT JOIN actors a ON r.did = a.did WHERE r.email= ?", nil, email).Scan(&repo).Error; err != nil {
		return nil, err
	}
	return &repo, nil
}

func (s *Server) getRepoActorByDid(ctx context.Context, did string) (*models.RepoActor, error) {
	var repo models.RepoActor
	if err := s.db.Raw(ctx, "SELECT r.*, a.* FROM repos r LEFT JOIN actors a ON r.did = a.did WHERE r.did = ?", nil, did).Scan(&repo).Error; err != nil {
		return nil, err
	}
	return &repo, nil
}
