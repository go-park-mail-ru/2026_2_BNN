package api

import (
	"context"
)

// AuthHandler описывает auth-часть StrictServerInterface.
type AuthHandler interface {
	SignUp(ctx context.Context, req SignUpRequestObject) (SignUpResponseObject, error)
	SignIn(ctx context.Context, req SignInRequestObject) (SignInResponseObject, error)
	Refresh(ctx context.Context, req RefreshRequestObject) (RefreshResponseObject, error)
	Logout(ctx context.Context, req LogoutRequestObject) (LogoutResponseObject, error)
	GetCurrentUser(ctx context.Context, req GetCurrentUserRequestObject) (GetCurrentUserResponseObject, error)
	UpdateAvatar(ctx context.Context, req UpdateAvatarRequestObject) (UpdateAvatarResponseObject, error)
}

// NotesHandler описывает notes-часть StrictServerInterface.
type NotesHandler interface {
	ListNotes(ctx context.Context, req ListNotesRequestObject) (ListNotesResponseObject, error)
	GetNote(ctx context.Context, req GetNoteRequestObject) (GetNoteResponseObject, error)
	CreateNote(ctx context.Context, req CreateNoteRequestObject) (CreateNoteResponseObject, error)
	UpdateNote(ctx context.Context, req UpdateNoteRequestObject) (UpdateNoteResponseObject, error)
	DeleteNote(ctx context.Context, req DeleteNoteRequestObject) (DeleteNoteResponseObject, error)
}

// Server реализует StrictServerInterface, делегируя в AuthHandler и NotesHandler.
type Server struct {
	auth  AuthHandler
	notes NotesHandler
}

func NewServer(auth AuthHandler, notes NotesHandler) *Server {
	return &Server{auth: auth, notes: notes}
}

// --- auth ---

func (s *Server) SignUp(ctx context.Context, req SignUpRequestObject) (SignUpResponseObject, error) {
	return s.auth.SignUp(ctx, req)
}

func (s *Server) SignIn(ctx context.Context, req SignInRequestObject) (SignInResponseObject, error) {
	return s.auth.SignIn(ctx, req)
}

func (s *Server) Refresh(ctx context.Context, req RefreshRequestObject) (RefreshResponseObject, error) {
	return s.auth.Refresh(ctx, req)
}

func (s *Server) Logout(ctx context.Context, req LogoutRequestObject) (LogoutResponseObject, error) {
	return s.auth.Logout(ctx, req)
}

func (s *Server) GetCurrentUser(ctx context.Context, req GetCurrentUserRequestObject) (GetCurrentUserResponseObject, error) {
	return s.auth.GetCurrentUser(ctx, req)
}

func (s *Server) UpdateAvatar(ctx context.Context, req UpdateAvatarRequestObject) (UpdateAvatarResponseObject, error) {
	return s.auth.UpdateAvatar(ctx, req)
}

// --- notes ---

func (s *Server) ListNotes(ctx context.Context, req ListNotesRequestObject) (ListNotesResponseObject, error) {
	return s.notes.ListNotes(ctx, req)
}

func (s *Server) GetNote(ctx context.Context, req GetNoteRequestObject) (GetNoteResponseObject, error) {
	return s.notes.GetNote(ctx, req)
}

func (s *Server) CreateNote(ctx context.Context, req CreateNoteRequestObject) (CreateNoteResponseObject, error) {
	return s.notes.CreateNote(ctx, req)
}

func (s *Server) UpdateNote(ctx context.Context, req UpdateNoteRequestObject) (UpdateNoteResponseObject, error) {
	return s.notes.UpdateNote(ctx, req)
}

func (s *Server) DeleteNote(ctx context.Context, req DeleteNoteRequestObject) (DeleteNoteResponseObject, error) {
	return s.notes.DeleteNote(ctx, req)
}
