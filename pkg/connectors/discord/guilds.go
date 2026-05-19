package discord

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

type guildsClient struct {
	client *discordClient
}

func (g *guildsClient) Get(ctx context.Context, guildID string) (*Guild, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if guildID == "" {
		return nil, fmt.Errorf("guildID cannot be empty")
	}

	path := fmt.Sprintf("/guilds/%s", guildID)

	respBody, err := g.client.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get guild: %w", err)
	}

	var guild Guild
	if err := json.Unmarshal(respBody, &guild); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &guild, nil
}

func (g *guildsClient) GetChannels(ctx context.Context, guildID string) ([]Channel, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if guildID == "" {
		return nil, fmt.Errorf("guildID cannot be empty")
	}

	path := fmt.Sprintf("/guilds/%s/channels", guildID)

	respBody, err := g.client.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get guild channels: %w", err)
	}

	var channels []Channel
	if err := json.Unmarshal(respBody, &channels); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return channels, nil
}

func (g *guildsClient) GetMembers(ctx context.Context, guildID string, params MemberListParams) ([]Member, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if guildID == "" {
		return nil, fmt.Errorf("guildID cannot be empty")
	}

	path := fmt.Sprintf("/guilds/%s/members", guildID)

	query := url.Values{}
	if params.Limit > 0 {
		query.Set("limit", strconv.Itoa(params.Limit))
	}
	if params.After != "" {
		query.Set("after", params.After)
	}

	if len(query) > 0 {
		path += "?" + query.Encode()
	}

	respBody, err := g.client.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get guild members: %w", err)
	}

	var members []Member
	if err := json.Unmarshal(respBody, &members); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return members, nil
}

func (g *guildsClient) GetMember(ctx context.Context, guildID, userID string) (*Member, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if guildID == "" {
		return nil, fmt.Errorf("guildID cannot be empty")
	}
	if userID == "" {
		return nil, fmt.Errorf("userID cannot be empty")
	}

	path := fmt.Sprintf("/guilds/%s/members/%s", guildID, userID)

	respBody, err := g.client.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get guild member: %w", err)
	}

	var member Member
	if err := json.Unmarshal(respBody, &member); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &member, nil
}

func (g *guildsClient) CreateChannel(ctx context.Context, guildID string, params *ChannelCreate) (*Channel, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if guildID == "" {
		return nil, fmt.Errorf("guildID cannot be empty")
	}
	if params == nil {
		return nil, fmt.Errorf("params cannot be nil")
	}
	if params.Name == "" {
		return nil, fmt.Errorf("channel name cannot be empty")
	}

	path := fmt.Sprintf("/guilds/%s/channels", guildID)

	respBody, err := g.client.doRequest(ctx, "POST", path, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create channel: %w", err)
	}

	var channel Channel
	if err := json.Unmarshal(respBody, &channel); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &channel, nil
}

func (g *guildsClient) GetRoles(ctx context.Context, guildID string) ([]Role, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if guildID == "" {
		return nil, fmt.Errorf("guildID cannot be empty")
	}

	path := fmt.Sprintf("/guilds/%s/roles", guildID)

	respBody, err := g.client.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get guild roles: %w", err)
	}

	var roles []Role
	if err := json.Unmarshal(respBody, &roles); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return roles, nil
}
