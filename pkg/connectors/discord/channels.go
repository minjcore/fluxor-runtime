package discord

import (
	"context"
	"encoding/json"
	"fmt"
)

type channelsClient struct {
	client *discordClient
}

func (c *channelsClient) Get(ctx context.Context, channelID string) (*Channel, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if channelID == "" {
		return nil, fmt.Errorf("channelID cannot be empty")
	}

	path := fmt.Sprintf("/channels/%s", channelID)

	respBody, err := c.client.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get channel: %w", err)
	}

	var channel Channel
	if err := json.Unmarshal(respBody, &channel); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &channel, nil
}

func (c *channelsClient) Modify(ctx context.Context, channelID string, params *ChannelModify) (*Channel, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if channelID == "" {
		return nil, fmt.Errorf("channelID cannot be empty")
	}
	if params == nil {
		return nil, fmt.Errorf("params cannot be nil")
	}

	path := fmt.Sprintf("/channels/%s", channelID)

	respBody, err := c.client.doRequest(ctx, "PATCH", path, params)
	if err != nil {
		return nil, fmt.Errorf("failed to modify channel: %w", err)
	}

	var channel Channel
	if err := json.Unmarshal(respBody, &channel); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &channel, nil
}

func (c *channelsClient) Delete(ctx context.Context, channelID string) error {
	if ctx == nil {
		return fmt.Errorf("context cannot be nil")
	}
	if channelID == "" {
		return fmt.Errorf("channelID cannot be empty")
	}

	path := fmt.Sprintf("/channels/%s", channelID)

	_, err := c.client.doRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return fmt.Errorf("failed to delete channel: %w", err)
	}

	return nil
}

func (c *channelsClient) CreateInvite(ctx context.Context, channelID string, params *InviteCreate) (*Invite, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if channelID == "" {
		return nil, fmt.Errorf("channelID cannot be empty")
	}

	path := fmt.Sprintf("/channels/%s/invites", channelID)

	var reqBody interface{}
	if params != nil {
		reqBody = params
	} else {
		reqBody = map[string]interface{}{}
	}

	respBody, err := c.client.doRequest(ctx, "POST", path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create invite: %w", err)
	}

	var invite Invite
	if err := json.Unmarshal(respBody, &invite); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &invite, nil
}
