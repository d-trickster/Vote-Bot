package tgclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"
)

const (
	tgHost = "api.telegram.org"

	methodSendMessage     = "sendMessage"
	methodGetUpdates      = "getUpdates"
	methodSetMyCommands   = "setMyCommands"
	methodEditMessageText = "editMessageText"
	methodGetChatAdmins   = "getChatAdministrators"

	scopeAllPrivate    = "all_private_chats"
	scopeAllGroupChats = "all_group_chats"
	scopeAllChatAdmins = "all_chat_administrators"
	// scopeChat          = "chat"
)

type Client struct {
	baseURL string
	client  http.Client
}

func NewClient(token string) *Client {
	return &Client{
		baseURL: "bot" + token,
		client:  http.Client{Timeout: time.Second * 5},
	}
}

func (c *Client) Updates(limit int, offset int) ([]Update, error) {
	q := url.Values{}
	q.Add("limit", strconv.Itoa(limit))
	if offset != 0 {
		q.Add("offset", strconv.Itoa(offset))
	}

	resp, err := c.doRequest(methodGetUpdates, q, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to do request: %w", err)
	}
	defer resp.Close()

	var upd UpdatesResponse
	if err = json.NewDecoder(resp).Decode(&upd); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	if !upd.Ok {
		return nil, fmt.Errorf("response is not ok: %d - %s", upd.ErrorCode, upd.Descr)
	}

	return upd.Updates, nil
}

func (c *Client) AnswerWithResult(msg *Message, text string) (*Message, error) {
	data, err := json.Marshal(SendMessageParams{
		ChatId:    msg.Chat.Id,
		ThreadId:  msg.ThreadId,
		Text:      text,
		ParseMode: "HTML",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %w", err)
	}
	body := bytes.NewBuffer(data)

	resp, err := c.doRequest(methodSendMessage, nil, body)
	if err != nil {
		return nil, fmt.Errorf("faield to send message: %w", err)
	}
	defer resp.Close()

	var res WithMessageResponse
	if err = json.NewDecoder(resp).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	if !res.Ok {
		return nil, fmt.Errorf("response is not ok: %d - %s", res.ErrorCode, res.Descr)
	}

	return &res.Message, nil
}

func (c *Client) EditMessage(chatID int64, messageID int64, text string, keyboard InlineKeyboardMarkup) error {
	data, err := json.Marshal(EditMessageParams{
		SendMessageParams: SendMessageParams{
			ChatId:    chatID,
			Text:      text,
			ParseMode: "HTML",
			Keyboard:  &keyboard,
		},
		MessageId: messageID,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}
	body := bytes.NewBuffer(data)

	resp, err := c.doRequest(methodEditMessageText, nil, body)
	if err != nil {
		return fmt.Errorf("faield to send message: %w", err)
	}

	var result CommonResponse
	if err = json.NewDecoder(resp).Decode(&result); err != nil {
		slog.Error("failed to decode EditMessage response: " + err.Error())
	}
	if !result.Ok {
		return fmt.Errorf("failed to edit message with code %d: %s", result.ErrorCode, result.Descr)
	}

	return nil
}

func (c *Client) SendInlineKeyboard(chatID int64, text string, keyboard InlineKeyboardMarkup) error {
	data, err := json.Marshal(SendMessageParams{
		ChatId:    chatID,
		Text:      text,
		ParseMode: "HTML",
		Keyboard:  &keyboard,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}
	body := bytes.NewBuffer(data)

	resp, err := c.doRequest(methodSendMessage, nil, body)
	if err != nil {
		return fmt.Errorf("faield to send message: %w", err)
	}

	var result CommonResponse
	if err = json.NewDecoder(resp).Decode(&result); err != nil {
		slog.Error("failed to decode SendInlineKeyboard response: " + err.Error())
	}
	if !result.Ok {
		return fmt.Errorf("failed to send inline keyboard with code %d: %s", result.ErrorCode, result.Descr)
	}

	return nil
}

func (c *Client) Answer(msg *Message, text string) error {
	data, err := json.Marshal(SendMessageParams{
		ChatId:    msg.Chat.Id,
		ThreadId:  msg.ThreadId,
		Text:      text,
		ParseMode: "HTML",
	})
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}
	body := bytes.NewBuffer(data)

	resp, err := c.doRequest(methodSendMessage, nil, body)
	if err != nil {
		return fmt.Errorf("faield to send message: %w", err)
	}

	var result CommonResponse
	if err = json.NewDecoder(resp).Decode(&result); err != nil {
		slog.Error("failed to decode Answer response: " + err.Error())
	}
	if !result.Ok {
		return fmt.Errorf("failed to answer with code %d: %s", result.ErrorCode, result.Descr)
	}

	return nil
}

func (c *Client) SetCommandsPrivate(commands [][]string) error {
	return c.setCommands(commands, CommandScope{Type: scopeAllPrivate})
}

// func (c *Client) SetCommandChat(commands [][]string, chatId int64) error {
// 	return c.setCommands(commands, CommandScope{Type: scopeChat, ChatId: chatId})
// }

func (c *Client) SetCommandsGroup(commands [][]string) error {
	return c.setCommands(commands, CommandScope{Type: scopeAllGroupChats})
}

func (c *Client) SetCommandsGroupAdmin(commands [][]string) error {
	return c.setCommands(commands, CommandScope{Type: scopeAllChatAdmins})
}

func (c *Client) setCommands(commands [][]string, scope CommandScope) error {
	params := SetCommandsParams{
		Commands: make([]Command, len(commands)),
		Scope:    scope,
	}
	for i := range commands {
		params.Commands[i].Cmd = commands[i][0]
		params.Commands[i].Descr = commands[i][1]
	}
	data, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("failed to marshal commands: %w", err)
	}

	body := bytes.NewBuffer(data)
	resp, err := c.doRequest(methodSetMyCommands, nil, body)
	if err != nil {
		return fmt.Errorf("failed to set commands: %w", err)
	}

	var result CommonResponse
	if err = json.NewDecoder(resp).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode SetCommands response: %w", err)
	}
	if !result.Ok {
		return fmt.Errorf("failed to set commands with code %d: %s", result.ErrorCode, result.Descr)
	}

	return nil
}

func (c *Client) ChatAdmins(chatId int64) ([]Admin, error) {
	q := url.Values{}
	q.Add("chat_id", fmt.Sprintf("%d", chatId))

	resp, err := c.doRequest(methodGetChatAdmins, q, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat admins: %w", err)
	}

	var result WithAdminsResponse
	if err := json.NewDecoder(resp).Decode(&result); err != nil {
		return nil, fmt.Errorf("faield to decode ChatAdmins response: %w", err)
	}
	if !result.Ok {
		return nil, fmt.Errorf("faield to get chat admins with code %d: %s", result.ErrorCode, result.Descr)
	}

	return result.Admins, nil
}

func (c *Client) doRequest(method string, query url.Values, body io.Reader) (io.ReadCloser, error) {
	u := url.URL{
		Scheme: "https",
		Host:   tgHost,
		Path:   path.Join(c.baseURL, method),
	}

	req, err := http.NewRequest(http.MethodPost, u.String(), body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-type", "application/json")
	if query != nil {
		req.URL.RawQuery = query.Encode()
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to do request: %w", err)
	}

	return resp.Body, nil
}
