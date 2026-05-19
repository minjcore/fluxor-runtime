package trello

import (
	"context"
	"encoding/json"
	"fmt"
)

type membersClient struct {
	client *trelloClient
}

func (m *membersClient) Get(ctx context.Context, memberID string) (*Member, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if memberID == "" {
		return nil, fmt.Errorf("memberID cannot be empty")
	}

	path := fmt.Sprintf("/members/%s", memberID)

	respBody, err := m.client.doRequest(ctx, "GET", path, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get member: %w", err)
	}

	var member Member
	if err := json.Unmarshal(respBody, &member); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &member, nil
}

func (m *membersClient) GetMe(ctx context.Context) (*Member, error) {
	return m.Get(ctx, "me")
}

func (m *membersClient) GetBoards(ctx context.Context, memberID string) ([]Board, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if memberID == "" {
		return nil, fmt.Errorf("memberID cannot be empty")
	}

	path := fmt.Sprintf("/members/%s/boards", memberID)

	respBody, err := m.client.doRequest(ctx, "GET", path, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get member boards: %w", err)
	}

	var boards []Board
	if err := json.Unmarshal(respBody, &boards); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return boards, nil
}
