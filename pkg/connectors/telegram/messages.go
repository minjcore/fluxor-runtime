package telegram

import (
	"context"
	"encoding/json"
	"fmt"
)

type messagesClient struct {
	client *telegramClient
}

func (m *messagesClient) Send(ctx context.Context, chatID int64, req *SendMessageRequest) (*Message, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	params := map[string]interface{}{
		"chat_id": chatID,
		"text":    req.Text,
	}

	if req.ParseMode != "" {
		params["parse_mode"] = req.ParseMode
	}
	if req.DisableWebPagePreview {
		params["disable_web_page_preview"] = req.DisableWebPagePreview
	}
	if req.DisableNotification {
		params["disable_notification"] = req.DisableNotification
	}
	if req.ReplyToMessageID != 0 {
		params["reply_to_message_id"] = req.ReplyToMessageID
	}
	if req.ReplyMarkup != nil {
		params["reply_markup"] = req.ReplyMarkup
	}

	result, err := m.client.doRequest(ctx, "sendMessage", params)
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	var msg Message
	if err := json.Unmarshal(result, &msg); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &msg, nil
}

func (m *messagesClient) SendPhoto(ctx context.Context, chatID int64, req *SendPhotoRequest) (*Message, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	params := map[string]interface{}{
		"chat_id": chatID,
		"photo":   req.Photo,
	}

	if req.Caption != "" {
		params["caption"] = req.Caption
	}
	if req.ParseMode != "" {
		params["parse_mode"] = req.ParseMode
	}
	if req.DisableNotification {
		params["disable_notification"] = req.DisableNotification
	}
	if req.ReplyToMessageID != 0 {
		params["reply_to_message_id"] = req.ReplyToMessageID
	}
	if req.ReplyMarkup != nil {
		params["reply_markup"] = req.ReplyMarkup
	}

	result, err := m.client.doRequest(ctx, "sendPhoto", params)
	if err != nil {
		return nil, fmt.Errorf("failed to send photo: %w", err)
	}

	var msg Message
	if err := json.Unmarshal(result, &msg); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &msg, nil
}

func (m *messagesClient) SendDocument(ctx context.Context, chatID int64, req *SendDocumentRequest) (*Message, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	params := map[string]interface{}{
		"chat_id":  chatID,
		"document": req.Document,
	}

	if req.Caption != "" {
		params["caption"] = req.Caption
	}
	if req.ParseMode != "" {
		params["parse_mode"] = req.ParseMode
	}
	if req.DisableNotification {
		params["disable_notification"] = req.DisableNotification
	}
	if req.ReplyToMessageID != 0 {
		params["reply_to_message_id"] = req.ReplyToMessageID
	}
	if req.ReplyMarkup != nil {
		params["reply_markup"] = req.ReplyMarkup
	}

	result, err := m.client.doRequest(ctx, "sendDocument", params)
	if err != nil {
		return nil, fmt.Errorf("failed to send document: %w", err)
	}

	var msg Message
	if err := json.Unmarshal(result, &msg); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &msg, nil
}

func (m *messagesClient) Edit(ctx context.Context, chatID int64, messageID int, text string) (*Message, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	params := map[string]interface{}{
		"chat_id":    chatID,
		"message_id": messageID,
		"text":       text,
	}

	result, err := m.client.doRequest(ctx, "editMessageText", params)
	if err != nil {
		return nil, fmt.Errorf("failed to edit message: %w", err)
	}

	var msg Message
	if err := json.Unmarshal(result, &msg); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &msg, nil
}

func (m *messagesClient) Delete(ctx context.Context, chatID int64, messageID int) error {
	if ctx == nil {
		return fmt.Errorf("context cannot be nil")
	}

	params := map[string]interface{}{
		"chat_id":    chatID,
		"message_id": messageID,
	}

	_, err := m.client.doRequest(ctx, "deleteMessage", params)
	if err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}

	return nil
}

func (m *messagesClient) Forward(ctx context.Context, chatID, fromChatID int64, messageID int) (*Message, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}

	params := map[string]interface{}{
		"chat_id":      chatID,
		"from_chat_id": fromChatID,
		"message_id":   messageID,
	}

	result, err := m.client.doRequest(ctx, "forwardMessage", params)
	if err != nil {
		return nil, fmt.Errorf("failed to forward message: %w", err)
	}

	var msg Message
	if err := json.Unmarshal(result, &msg); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &msg, nil
}
