package mcp

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"bnn/internal/models"

	"github.com/samber/lo"
)

const (
	blockHeading   = "heading"
	blockSubtitle  = "subtitle"
	blockBulleted  = "bulleted"
	blockNumbered  = "numbered"
	blockChecklist = "checklist"
	blockQuote     = "quote"
	blockFile      = "file"
	blockImage     = "image"

	untitled  = "Без названия"
	noNotes   = "Заметок нет"
	emptyNote = "Заметка пустая"
)

type blockContent struct {
	Type    string `json:"type"`
	Text    string `json:"text"`
	Checked bool   `json:"checked"`
}

func formatNotes(list []models.Note) string {
	if len(list) == 0 {
		return noNotes
	}

	var b strings.Builder
	for _, n := range list {
		b.WriteString("- " + lo.CoalesceOrEmpty(n.Title, untitled) + " (id: " + n.ID.String())
		if n.ParentNoteID != nil {
			b.WriteString(", родитель: " + n.ParentNoteID.String())
		}
		b.WriteString(", изменена: " + n.UpdatedAt.Format(time.DateOnly) + ")\n")
	}

	return b.String()
}

func formatNote(n models.Note) string {
	var b strings.Builder
	b.WriteString("# " + lo.CoalesceOrEmpty(n.Title, untitled) + "\n")

	if len(n.Tags) > 0 {
		names := lo.Map(n.Tags, func(t models.Tag, _ int) string { return t.Name })
		b.WriteString("Теги: " + strings.Join(names, ", ") + "\n")
	}
	b.WriteString("\n")

	if len(n.Blocks) == 0 {
		b.WriteString(emptyNote)
		return b.String()
	}

	number := 0
	for _, block := range n.Blocks {
		var content blockContent
		if err := json.Unmarshal(block.Content, &content); err != nil {
			continue
		}

		number = lo.Ternary(content.Type == blockNumbered, number+1, 0)
		b.WriteString(formatBlock(block, content, number) + "\n")
	}

	return b.String()
}

func formatBlock(block models.Block, content blockContent, number int) string {
	switch content.Type {
	case blockHeading:
		return "## " + content.Text
	case blockSubtitle:
		return "### " + content.Text
	case blockBulleted:
		return "• " + content.Text
	case blockNumbered:
		return strconv.Itoa(number) + ". " + content.Text
	case blockChecklist:
		return lo.Ternary(content.Checked, "[x] ", "[ ] ") + content.Text + "  (block: " + block.ID.String() + ")"
	case blockQuote:
		return "> " + content.Text
	case blockFile:
		return "[файл] " + content.Text
	case blockImage:
		return "[картинка] " + content.Text
	default:
		return content.Text
	}
}
