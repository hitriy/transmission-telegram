package bot

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/hitriy/transmission-telegram/rutracker"
	"github.com/pyed/transmission"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

var rutrackerTopicRe = regexp.MustCompile(`(?i)rutracker\.(org|net|nl)/forum/viewtopic\.php\?t=(\d+)`)

type RuTrackerHandler struct {
	Client             *rutracker.Client
	TBot               *tgbotapi.BotAPI
	Trans              *transmission.TransmissionClient
	Sessions           *SessionStore
	Logger             *log.Logger
	EnsureTransmission func() error
}

func (h *RuTrackerHandler) ExtractTopicID(text string) (string, bool) {
	match := rutrackerTopicRe.FindStringSubmatch(text)
	if len(match) < 3 {
		return "", false
	}
	return match[2], true
}

func (h *RuTrackerHandler) HandleLink(ud tgbotapi.Update) {
	text := strings.TrimSpace(ud.Message.Text)
	id, ok := h.ExtractTopicID(text)
	if !ok {
		h.sendPlain("RuTracker: invalid topic link", ud.Message.Chat.ID)
		return
	}

	h.sendPlain("RuTracker: fetching magnet…", ud.Message.Chat.ID)

	thread, err := h.Client.GetThread(id)
	if err != nil {
		h.sendPlain("RuTracker: "+err.Error(), ud.Message.Chat.ID)
		return
	}

	h.addMagnet(ud.Message.Chat.ID, thread.Magnet, thread.Title)
}

func (h *RuTrackerHandler) HandleSearch(ud tgbotapi.Update) {
	query := strings.TrimSpace(ud.Message.Text)
	if query == "" {
		return
	}

	h.sendPlain("RuTracker: searching…", ud.Message.Chat.ID)

	results, err := h.Client.Search(query)
	if err != nil {
		h.sendPlain("RuTracker: "+err.Error(), ud.Message.Chat.ID)
		return
	}

	if len(results) == 0 {
		h.sendPlain(fmt.Sprintf("RuTracker: no results for \"%s\"", query), ud.Message.Chat.ID)
		return
	}

	session := SearchSession{
		Query:   query,
		Results: results,
		Page:    0,
		ChatID:  ud.Message.Chat.ID,
	}

	text, keyboard := h.buildResultsView(session)
	msgID := h.sendWithKeyboard(text, ud.Message.Chat.ID, keyboard)
	session.MessageID = msgID
	h.Sessions.Set(msgID, session)
}

func (h *RuTrackerHandler) HandleCallback(cq *tgbotapi.CallbackQuery) {
	if cq.Message == nil {
		h.answerCallback(cq.ID, "")
		return
	}

	msgID := cq.Message.MessageID
	session, ok := h.Sessions.Get(msgID)
	if !ok {
		h.answerCallback(cq.ID, "Session expired")
		return
	}

	data := cq.Data
	switch {
	case strings.HasPrefix(data, "rt:p:"):
		delta, _ := strconv.Atoi(strings.TrimPrefix(data, "rt:p:"))
		pages := totalPages(len(session.Results))
		session.Page += delta
		if session.Page < 0 {
			session.Page = 0
		}
		if session.Page >= pages {
			session.Page = pages - 1
		}
		session.SelectedID = ""
		h.Sessions.Set(msgID, session)
		text, keyboard := h.buildResultsView(session)
		h.editWithKeyboard(session.ChatID, msgID, text, keyboard)
		h.answerCallback(cq.ID, "")

	case strings.HasPrefix(data, "rt:s:"):
		idx, err := strconv.Atoi(strings.TrimPrefix(data, "rt:s:"))
		if err != nil || idx < 0 || idx >= pageSize {
			h.answerCallback(cq.ID, "Invalid selection")
			return
		}
		items := pageItems(session.Results, session.Page)
		if idx >= len(items) {
			h.answerCallback(cq.ID, "Invalid selection")
			return
		}
		selected := items[idx]
		session.SelectedID = selected.ID
		h.Sessions.Set(msgID, session)

		thread, err := h.Client.GetThread(selected.ID)
		if err != nil {
			h.answerCallback(cq.ID, err.Error())
			return
		}

		text, keyboard := h.buildDetailView(selected, thread)
		h.editWithKeyboard(session.ChatID, msgID, text, keyboard)
		h.answerCallback(cq.ID, "")

	case data == "rt:d":
		if session.SelectedID == "" {
			h.answerCallback(cq.ID, "Nothing selected")
			return
		}
		thread, err := h.Client.GetThread(session.SelectedID)
		if err != nil {
			h.answerCallback(cq.ID, err.Error())
			return
		}
		if err := h.addMagnet(session.ChatID, thread.Magnet, thread.Title); err != nil {
			h.answerCallback(cq.ID, err.Error())
			return
		}
		h.answerCallback(cq.ID, "Added to Transmission")

	case data == "rt:b":
		session.SelectedID = ""
		h.Sessions.Set(msgID, session)
		text, keyboard := h.buildResultsView(session)
		h.editWithKeyboard(session.ChatID, msgID, text, keyboard)
		h.answerCallback(cq.ID, "")

	default:
		h.answerCallback(cq.ID, "")
	}
}

func (h *RuTrackerHandler) buildResultsView(session SearchSession) (string, tgbotapi.InlineKeyboardMarkup) {
	items := pageItems(session.Results, session.Page)
	pages := totalPages(len(session.Results))
	pageNum := session.Page + 1

	var b strings.Builder
	fmt.Fprintf(&b, "RuTracker: \"%s\" — page %d/%d (%d found)\n\n", session.Query, pageNum, pages, len(session.Results))

	for i, t := range items {
		title := truncate(t.Title, 80)
		fmt.Fprintf(&b, "%d. %s [%s] S:%d L:%d\n", i+1, title, t.FormattedSize(), t.Seeds, t.Leeches)
	}

	keyboard := h.buildResultsKeyboard(session, len(items))
	return b.String(), keyboard
}

func (h *RuTrackerHandler) buildDetailView(t rutracker.Torrent, thread rutracker.Thread) (string, tgbotapi.InlineKeyboardMarkup) {
	title := thread.Title
	if title == "" {
		title = t.Title
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", title)
	fmt.Fprintf(&b, "%s | Seeds: %d | Leeches: %d\n\n", t.FormattedSize(), t.Seeds, t.Leeches)
	if thread.Description != "" {
		b.WriteString(thread.Description)
	} else {
		b.WriteString("(no description)")
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Download", "rt:d"),
			tgbotapi.NewInlineKeyboardButtonData("Back", "rt:b"),
		),
	)
	return b.String(), keyboard
}

func (h *RuTrackerHandler) buildResultsKeyboard(session SearchSession, itemCount int) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton

	var numRow []tgbotapi.InlineKeyboardButton
	for i := 0; i < itemCount; i++ {
		label := strconv.Itoa(i + 1)
		numRow = append(numRow, tgbotapi.NewInlineKeyboardButtonData(label, fmt.Sprintf("rt:s:%d", i)))
		if len(numRow) == 5 {
			rows = append(rows, numRow)
			numRow = nil
		}
	}
	if len(numRow) > 0 {
		rows = append(rows, numRow)
	}

	pages := totalPages(len(session.Results))
	var navRow []tgbotapi.InlineKeyboardButton
	if session.Page > 0 {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("◀ Prev", "rt:p:-1"))
	}
	if session.Page < pages-1 {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("Next ▶", "rt:p:1"))
	}
	if len(navRow) > 0 {
		rows = append(rows, navRow)
	}

	return tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func (h *RuTrackerHandler) addMagnet(chatID int64, magnet, name string) error {
	if h.Trans == nil && h.EnsureTransmission != nil {
		if err := h.EnsureTransmission(); err != nil {
			h.sendPlain("Transmission: "+err.Error(), chatID)
			return err
		}
	}
	if h.Trans == nil {
		err := fmt.Errorf("transmission is not available")
		h.sendPlain(err.Error(), chatID)
		return err
	}

	cmd := transmission.NewAddCmdByURL(magnet)
	torrent, err := h.Trans.ExecuteAddCommand(cmd)
	if err != nil {
		h.sendPlain("*add:* "+err.Error(), chatID)
		return err
	}
	if torrent.Name == "" {
		err := fmt.Errorf("error adding torrent")
		h.sendPlain("*add:* "+err.Error(), chatID)
		return err
	}
	display := torrent.Name
	if name != "" {
		display = name
	}
	h.sendPlain(fmt.Sprintf("*Added:* <%d> %s", torrent.ID, display), chatID)
	return nil
}

func (h *RuTrackerHandler) sendPlain(text string, chatID int64) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.DisableWebPagePreview = true
	if _, err := h.TBot.Send(msg); err != nil {
		h.Logger.Printf("[ERROR] RuTracker send: %s", err)
	}
}

func (h *RuTrackerHandler) sendWithKeyboard(text string, chatID int64, keyboard tgbotapi.InlineKeyboardMarkup) int {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.DisableWebPagePreview = true
	msg.ReplyMarkup = keyboard
	resp, err := h.TBot.Send(msg)
	if err != nil {
		h.Logger.Printf("[ERROR] RuTracker send: %s", err)
		return 0
	}
	return resp.MessageID
}

func (h *RuTrackerHandler) editWithKeyboard(chatID int64, msgID int, text string, keyboard tgbotapi.InlineKeyboardMarkup) {
	edit := tgbotapi.NewEditMessageText(chatID, msgID, text)
	edit.DisableWebPagePreview = true
	edit.ReplyMarkup = &keyboard
	if _, err := h.TBot.Send(edit); err != nil {
		h.Logger.Printf("[ERROR] RuTracker edit: %s", err)
	}
}

func (h *RuTrackerHandler) answerCallback(callbackID, text string) {
	cb := tgbotapi.NewCallback(callbackID, text)
	if _, err := h.TBot.AnswerCallbackQuery(cb); err != nil {
		h.Logger.Printf("[ERROR] RuTracker callback: %s", err)
	}
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-1]) + "…"
}
