package handler

import "context"

type RepoMgr interface {
	RegisterUser(ctx context.Context, userID int64, userName string) error
}

type Handler struct {
	repo RepoMgr
}

func Init(storage RepoMgr) *Handler {
	return &Handler{repo: storage}
}

func (h *Handler) RegisterUser(ctx context.Context, userID int64, userName string) error {
	return h.repo.RegisterUser(ctx, userID, userName)
}
