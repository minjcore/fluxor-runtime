package trello

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

type listsClient struct {
	client *trelloClient
}

func (l *listsClient) Get(ctx context.Context, listID string) (*List, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if listID == "" {
		return nil, fmt.Errorf("listID cannot be empty")
	}

	path := fmt.Sprintf("/lists/%s", listID)

	respBody, err := l.client.doRequest(ctx, "GET", path, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get list: %w", err)
	}

	var list List
	if err := json.Unmarshal(respBody, &list); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &list, nil
}

func (l *listsClient) Create(ctx context.Context, params *CreateListParams) (*List, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if params == nil {
		return nil, fmt.Errorf("params cannot be nil")
	}
	if params.Name == "" {
		return nil, fmt.Errorf("name cannot be empty")
	}
	if params.IDBoard == "" {
		return nil, fmt.Errorf("idBoard cannot be empty")
	}

	query := url.Values{}
	query.Set("name", params.Name)
	query.Set("idBoard", params.IDBoard)
	if params.Pos != "" {
		query.Set("pos", params.Pos)
	}

	respBody, err := l.client.doRequest(ctx, "POST", "/lists", nil, query)
	if err != nil {
		return nil, fmt.Errorf("failed to create list: %w", err)
	}

	var list List
	if err := json.Unmarshal(respBody, &list); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &list, nil
}

func (l *listsClient) Update(ctx context.Context, listID string, params *UpdateListParams) (*List, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if listID == "" {
		return nil, fmt.Errorf("listID cannot be empty")
	}

	query := url.Values{}
	if params != nil {
		if params.Name != "" {
			query.Set("name", params.Name)
		}
		if params.Closed != nil {
			query.Set("closed", fmt.Sprintf("%t", *params.Closed))
		}
		if params.Pos != "" {
			query.Set("pos", params.Pos)
		}
	}

	path := fmt.Sprintf("/lists/%s", listID)

	respBody, err := l.client.doRequest(ctx, "PUT", path, nil, query)
	if err != nil {
		return nil, fmt.Errorf("failed to update list: %w", err)
	}

	var list List
	if err := json.Unmarshal(respBody, &list); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &list, nil
}

func (l *listsClient) Archive(ctx context.Context, listID string) (*List, error) {
	closed := true
	return l.Update(ctx, listID, &UpdateListParams{Closed: &closed})
}

func (l *listsClient) GetCards(ctx context.Context, listID string) ([]Card, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if listID == "" {
		return nil, fmt.Errorf("listID cannot be empty")
	}

	path := fmt.Sprintf("/lists/%s/cards", listID)

	respBody, err := l.client.doRequest(ctx, "GET", path, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get list cards: %w", err)
	}

	var cards []Card
	if err := json.Unmarshal(respBody, &cards); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return cards, nil
}

func (l *listsClient) MoveAllCards(ctx context.Context, listID, targetListID string) error {
	if ctx == nil {
		return fmt.Errorf("context cannot be nil")
	}
	if listID == "" {
		return fmt.Errorf("listID cannot be empty")
	}
	if targetListID == "" {
		return fmt.Errorf("targetListID cannot be empty")
	}

	query := url.Values{}
	query.Set("idBoard", "")  // Will be determined by target list
	query.Set("idList", targetListID)

	path := fmt.Sprintf("/lists/%s/moveAllCards", listID)

	_, err := l.client.doRequest(ctx, "POST", path, nil, query)
	if err != nil {
		return fmt.Errorf("failed to move cards: %w", err)
	}

	return nil
}
