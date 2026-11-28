package mcp

import (
	"context"
	"net/http"

	"bnn/internal/pkg/auth"
	"bnn/internal/pkg/notes"

	sdkauth "github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const maxRequestBodySize = 1 << 20

func NewHandler(authUC auth.Usecase, notesUC notes.Usecase) http.Handler {
	server := NewServer(notesUC)

	mcpHandler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{
		Stateless:           true,
		JSONResponse:        true,
		MaxRequestBodyBytes: maxRequestBodySize,
	})

	authMiddleware := sdkauth.RequireBearerToken(verifyToken(authUC), &sdkauth.RequireBearerTokenOptions{
		AllowMissingExpiration: true,
	})

	return authMiddleware(mcpHandler)
}

func verifyToken(authUC auth.Usecase) sdkauth.TokenVerifier {
	return func(ctx context.Context, token string, _ *http.Request) (*sdkauth.TokenInfo, error) {
		userID, err := authUC.Authenticate(ctx, token)
		if err != nil {
			return nil, sdkauth.ErrInvalidToken
		}

		return &sdkauth.TokenInfo{UserID: userID.String()}, nil
	}
}
