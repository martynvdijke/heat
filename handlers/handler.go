package handlers

import "heat/app"

type Handler struct {
	S *app.Server
}

func New(s *app.Server) *Handler {
	return &Handler{S: s}
}
