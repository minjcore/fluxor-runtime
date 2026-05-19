package trello

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

type boardsClient struct {
	client *trelloClient
}

func (b *boardsClient) Get(ctx context.Context, boardID string) (*Board, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if boardID == "" {
		return nil, fmt.Errorf("boardID cannot be empty")
	}

	path := fmt.Sprintf("/boards/%s", boardID)

	respBody, err := b.client.doRequest(ctx, "GET", path, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get board: %w", err)
	}

	var board Board
	if err := json.Unmarshal(respBody, &board); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &board, nil
}

func (b *boardsClient) Create(ctx context.Context, params *CreateBoardParams) (*Board, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if params == nil {
		return nil, fmt.Errorf("params cannot be nil")
	}
	if params.Name == "" {
		return nil, fmt.Errorf("name cannot be empty")
	}

	query := url.Values{}
	query.Set("name", params.Name)
	if params.Desc != "" {
		query.Set("desc", params.Desc)
	}
	if params.IDOrganization != "" {
		query.Set("idOrganization", params.IDOrganization)
	}

	respBody, err := b.client.doRequest(ctx, "POST", "/boards", nil, query)
	if err != nil {
		return nil, fmt.Errorf("failed to create board: %w", err)
	}

	var board Board
	if err := json.Unmarshal(respBody, &board); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &board, nil
}

func (b *boardsClient) Update(ctx context.Context, boardID string, params *UpdateBoardParams) (*Board, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if boardID == "" {
		return nil, fmt.Errorf("boardID cannot be empty")
	}

	query := url.Values{}
	if params != nil {
		if params.Name != "" {
			query.Set("name", params.Name)
		}
		if params.Desc != "" {
			query.Set("desc", params.Desc)
		}
		if params.Closed != nil {
			query.Set("closed", fmt.Sprintf("%t", *params.Closed))
		}
	}

	path := fmt.Sprintf("/boards/%s", boardID)

	respBody, err := b.client.doRequest(ctx, "PUT", path, nil, query)
	if err != nil {
		return nil, fmt.Errorf("failed to update board: %w", err)
	}

	var board Board
	if err := json.Unmarshal(respBody, &board); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &board, nil
}

func (b *boardsClient) Delete(ctx context.Context, boardID string) error {
	if ctx == nil {
		return fmt.Errorf("context cannot be nil")
	}
	if boardID == "" {
		return fmt.Errorf("boardID cannot be empty")
	}

	path := fmt.Sprintf("/boards/%s", boardID)

	_, err := b.client.doRequest(ctx, "DELETE", path, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to delete board: %w", err)
	}

	return nil
}

func (b *boardsClient) GetLists(ctx context.Context, boardID string) ([]List, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if boardID == "" {
		return nil, fmt.Errorf("boardID cannot be empty")
	}

	path := fmt.Sprintf("/boards/%s/lists", boardID)

	respBody, err := b.client.doRequest(ctx, "GET", path, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get board lists: %w", err)
	}

	var lists []List
	if err := json.Unmarshal(respBody, &lists); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return lists, nil
}

func (b *boardsClient) GetCards(ctx context.Context, boardID string) ([]Card, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if boardID == "" {
		return nil, fmt.Errorf("boardID cannot be empty")
	}

	path := fmt.Sprintf("/boards/%s/cards", boardID)

	respBody, err := b.client.doRequest(ctx, "GET", path, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get board cards: %w", err)
	}

	var cards []Card
	if err := json.Unmarshal(respBody, &cards); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return cards, nil
}

func (b *boardsClient) GetMembers(ctx context.Context, boardID string) ([]Member, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if boardID == "" {
		return nil, fmt.Errorf("boardID cannot be empty")
	}

	path := fmt.Sprintf("/boards/%s/members", boardID)

	respBody, err := b.client.doRequest(ctx, "GET", path, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get board members: %w", err)
	}

	var members []Member
	if err := json.Unmarshal(respBody, &members); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return members, nil
}
