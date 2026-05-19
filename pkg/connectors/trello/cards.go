package trello

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

type cardsClient struct {
	client *trelloClient
}

func (c *cardsClient) Get(ctx context.Context, cardID string) (*Card, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if cardID == "" {
		return nil, fmt.Errorf("cardID cannot be empty")
	}

	path := fmt.Sprintf("/cards/%s", cardID)

	respBody, err := c.client.doRequest(ctx, "GET", path, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get card: %w", err)
	}

	var card Card
	if err := json.Unmarshal(respBody, &card); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &card, nil
}

func (c *cardsClient) Create(ctx context.Context, params *CreateCardParams) (*Card, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if params == nil {
		return nil, fmt.Errorf("params cannot be nil")
	}
	if params.Name == "" {
		return nil, fmt.Errorf("name cannot be empty")
	}
	if params.IDList == "" {
		return nil, fmt.Errorf("idList cannot be empty")
	}

	query := url.Values{}
	query.Set("name", params.Name)
	query.Set("idList", params.IDList)
	if params.Desc != "" {
		query.Set("desc", params.Desc)
	}
	if params.Pos != "" {
		query.Set("pos", params.Pos)
	}
	if params.Due != "" {
		query.Set("due", params.Due)
	}
	if len(params.IDMembers) > 0 {
		query.Set("idMembers", strings.Join(params.IDMembers, ","))
	}
	if len(params.IDLabels) > 0 {
		query.Set("idLabels", strings.Join(params.IDLabels, ","))
	}

	respBody, err := c.client.doRequest(ctx, "POST", "/cards", nil, query)
	if err != nil {
		return nil, fmt.Errorf("failed to create card: %w", err)
	}

	var card Card
	if err := json.Unmarshal(respBody, &card); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &card, nil
}

func (c *cardsClient) Update(ctx context.Context, cardID string, params *UpdateCardParams) (*Card, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if cardID == "" {
		return nil, fmt.Errorf("cardID cannot be empty")
	}

	query := url.Values{}
	if params != nil {
		if params.Name != "" {
			query.Set("name", params.Name)
		}
		if params.Desc != "" {
			query.Set("desc", params.Desc)
		}
		if params.IDList != "" {
			query.Set("idList", params.IDList)
		}
		if params.Closed != nil {
			query.Set("closed", fmt.Sprintf("%t", *params.Closed))
		}
		if params.Pos != "" {
			query.Set("pos", params.Pos)
		}
		if params.Due != "" {
			query.Set("due", params.Due)
		}
		if params.DueComplete != nil {
			query.Set("dueComplete", fmt.Sprintf("%t", *params.DueComplete))
		}
	}

	path := fmt.Sprintf("/cards/%s", cardID)

	respBody, err := c.client.doRequest(ctx, "PUT", path, nil, query)
	if err != nil {
		return nil, fmt.Errorf("failed to update card: %w", err)
	}

	var card Card
	if err := json.Unmarshal(respBody, &card); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &card, nil
}

func (c *cardsClient) Delete(ctx context.Context, cardID string) error {
	if ctx == nil {
		return fmt.Errorf("context cannot be nil")
	}
	if cardID == "" {
		return fmt.Errorf("cardID cannot be empty")
	}

	path := fmt.Sprintf("/cards/%s", cardID)

	_, err := c.client.doRequest(ctx, "DELETE", path, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to delete card: %w", err)
	}

	return nil
}

func (c *cardsClient) Archive(ctx context.Context, cardID string) (*Card, error) {
	closed := true
	return c.Update(ctx, cardID, &UpdateCardParams{Closed: &closed})
}

func (c *cardsClient) Move(ctx context.Context, cardID, listID string) (*Card, error) {
	return c.Update(ctx, cardID, &UpdateCardParams{IDList: listID})
}

func (c *cardsClient) AddComment(ctx context.Context, cardID, text string) (*Action, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if cardID == "" {
		return nil, fmt.Errorf("cardID cannot be empty")
	}
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	query := url.Values{}
	query.Set("text", text)

	path := fmt.Sprintf("/cards/%s/actions/comments", cardID)

	respBody, err := c.client.doRequest(ctx, "POST", path, nil, query)
	if err != nil {
		return nil, fmt.Errorf("failed to add comment: %w", err)
	}

	var action Action
	if err := json.Unmarshal(respBody, &action); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &action, nil
}

func (c *cardsClient) AddLabel(ctx context.Context, cardID, labelID string) error {
	if ctx == nil {
		return fmt.Errorf("context cannot be nil")
	}
	if cardID == "" {
		return fmt.Errorf("cardID cannot be empty")
	}
	if labelID == "" {
		return fmt.Errorf("labelID cannot be empty")
	}

	query := url.Values{}
	query.Set("value", labelID)

	path := fmt.Sprintf("/cards/%s/idLabels", cardID)

	_, err := c.client.doRequest(ctx, "POST", path, nil, query)
	if err != nil {
		return fmt.Errorf("failed to add label: %w", err)
	}

	return nil
}

func (c *cardsClient) RemoveLabel(ctx context.Context, cardID, labelID string) error {
	if ctx == nil {
		return fmt.Errorf("context cannot be nil")
	}
	if cardID == "" {
		return fmt.Errorf("cardID cannot be empty")
	}
	if labelID == "" {
		return fmt.Errorf("labelID cannot be empty")
	}

	path := fmt.Sprintf("/cards/%s/idLabels/%s", cardID, labelID)

	_, err := c.client.doRequest(ctx, "DELETE", path, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to remove label: %w", err)
	}

	return nil
}

func (c *cardsClient) AddMember(ctx context.Context, cardID, memberID string) error {
	if ctx == nil {
		return fmt.Errorf("context cannot be nil")
	}
	if cardID == "" {
		return fmt.Errorf("cardID cannot be empty")
	}
	if memberID == "" {
		return fmt.Errorf("memberID cannot be empty")
	}

	query := url.Values{}
	query.Set("value", memberID)

	path := fmt.Sprintf("/cards/%s/idMembers", cardID)

	_, err := c.client.doRequest(ctx, "POST", path, nil, query)
	if err != nil {
		return fmt.Errorf("failed to add member: %w", err)
	}

	return nil
}

func (c *cardsClient) RemoveMember(ctx context.Context, cardID, memberID string) error {
	if ctx == nil {
		return fmt.Errorf("context cannot be nil")
	}
	if cardID == "" {
		return fmt.Errorf("cardID cannot be empty")
	}
	if memberID == "" {
		return fmt.Errorf("memberID cannot be empty")
	}

	path := fmt.Sprintf("/cards/%s/idMembers/%s", cardID, memberID)

	_, err := c.client.doRequest(ctx, "DELETE", path, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to remove member: %w", err)
	}

	return nil
}

func (c *cardsClient) GetChecklists(ctx context.Context, cardID string) ([]Checklist, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if cardID == "" {
		return nil, fmt.Errorf("cardID cannot be empty")
	}

	path := fmt.Sprintf("/cards/%s/checklists", cardID)

	respBody, err := c.client.doRequest(ctx, "GET", path, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get checklists: %w", err)
	}

	var checklists []Checklist
	if err := json.Unmarshal(respBody, &checklists); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return checklists, nil
}

func (c *cardsClient) AddChecklist(ctx context.Context, cardID string, params *CreateChecklistParams) (*Checklist, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if cardID == "" {
		return nil, fmt.Errorf("cardID cannot be empty")
	}
	if params == nil {
		return nil, fmt.Errorf("params cannot be nil")
	}
	if params.Name == "" {
		return nil, fmt.Errorf("name cannot be empty")
	}

	query := url.Values{}
	query.Set("name", params.Name)
	if params.IDChecklistSource != "" {
		query.Set("idChecklistSource", params.IDChecklistSource)
	}
	if params.Pos != "" {
		query.Set("pos", params.Pos)
	}

	path := fmt.Sprintf("/cards/%s/checklists", cardID)

	respBody, err := c.client.doRequest(ctx, "POST", path, nil, query)
	if err != nil {
		return nil, fmt.Errorf("failed to add checklist: %w", err)
	}

	var checklist Checklist
	if err := json.Unmarshal(respBody, &checklist); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &checklist, nil
}
