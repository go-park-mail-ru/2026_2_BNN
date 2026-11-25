package mcp

import (
	"context"

	"bnn/internal/pkg/notes"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	serverName    = "bnn"
	serverVersion = "1.0.0"
)

func NewServer(notesUC notes.Usecase) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: serverName, Version: serverVersion}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "ping",
		Description: "Проверка",
	}, ping)

	registerNoteTools(server, notesUC)

	return server
}

func ping(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "pong"}},
	}, nil, nil
}
