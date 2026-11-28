package mcp

import (
	"context"

	"bnn/internal/pkg/notes"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	serverName    = "bnn"
	serverVersion = "1.0.0"

	serverInstructions = `BNN — сервис заметок пользователя.
Заметки состоят из блоков: заголовки, текст, списки, цитаты, чек-листы, файлы.
Заметки могут быть вложенными: у подзаметки указан id родителя.
Чтобы найти заметку, вызови list_notes, затем прочитай её через get_note.
Пункты чек-листа показаны как [ ] (не сделано) и [x] (сделано).
Отвечай пользователю по содержимому заметок, не выдумывай то, чего в них нет.
Меняй заметки только тогда, когда пользователь прямо об этом попросил.`
)

func NewServer(notesUC notes.Usecase) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: serverName, Version: serverVersion},
		&mcp.ServerOptions{Instructions: serverInstructions})

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
