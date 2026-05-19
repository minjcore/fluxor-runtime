package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type documentsClient struct {
	client *elasticsearchClient
}

func (d *documentsClient) Index(ctx context.Context, index, id string, document interface{}) error {
	if ctx == nil {
		return fmt.Errorf("context cannot be nil")
	}
	if index == "" {
		return fmt.Errorf("index name cannot be empty")
	}

	var path string
	if id != "" {
		path = fmt.Sprintf("/%s/_doc/%s", index, id)
	} else {
		path = fmt.Sprintf("/%s/_doc", index)
	}

	_, err := d.client.doRequest(ctx, "POST", path, document)
	return err
}

func (d *documentsClient) Get(ctx context.Context, index, id string) (map[string]interface{}, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if index == "" {
		return nil, fmt.Errorf("index name cannot be empty")
	}
	if id == "" {
		return nil, fmt.Errorf("document ID cannot be empty")
	}

	path := fmt.Sprintf("/%s/_doc/%s", index, id)
	respBody, err := d.client.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract _source if present
	if source, ok := result["_source"].(map[string]interface{}); ok {
		return source, nil
	}

	return result, nil
}

func (d *documentsClient) Delete(ctx context.Context, index, id string) error {
	if ctx == nil {
		return fmt.Errorf("context cannot be nil")
	}
	if index == "" {
		return fmt.Errorf("index name cannot be empty")
	}
	if id == "" {
		return fmt.Errorf("document ID cannot be empty")
	}

	path := fmt.Sprintf("/%s/_doc/%s", index, id)
	_, err := d.client.doRequest(ctx, "DELETE", path, nil)
	return err
}

func (d *documentsClient) Exists(ctx context.Context, index, id string) (bool, error) {
	if ctx == nil {
		return false, fmt.Errorf("context cannot be nil")
	}
	if index == "" {
		return false, fmt.Errorf("index name cannot be empty")
	}
	if id == "" {
		return false, fmt.Errorf("document ID cannot be empty")
	}

	path := fmt.Sprintf("/%s/_doc/%s", index, id)
	_, err := d.client.doRequest(ctx, "HEAD", path, nil)
	if err != nil {
		if esErr, ok := err.(*ElasticsearchError); ok && esErr.Status == 404 {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (d *documentsClient) Update(ctx context.Context, index, id string, doc map[string]interface{}) error {
	if ctx == nil {
		return fmt.Errorf("context cannot be nil")
	}
	if index == "" {
		return fmt.Errorf("index name cannot be empty")
	}
	if id == "" {
		return fmt.Errorf("document ID cannot be empty")
	}

	path := fmt.Sprintf("/%s/_update/%s", index, id)
	updateBody := map[string]interface{}{
		"doc": doc,
	}

	_, err := d.client.doRequest(ctx, "POST", path, updateBody)
	return err
}

func (d *documentsClient) Bulk(ctx context.Context, operations []BulkOperation) (*BulkResponse, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if len(operations) == 0 {
		return nil, fmt.Errorf("operations cannot be empty")
	}

	// Build bulk request body (NDJSON format)
	var buf bytes.Buffer
	for _, op := range operations {
		// Action line
		action := map[string]interface{}{
			strings.ToLower(op.Operation): map[string]interface{}{
				"_index": op.Index,
			},
		}
		if op.ID != "" {
			action[strings.ToLower(op.Operation)].(map[string]interface{})["_id"] = op.ID
		}

		actionJSON, _ := json.Marshal(action)
		buf.Write(actionJSON)
		buf.WriteString("\n")

		// Document line (for index, create, update)
		if op.Operation == "index" || op.Operation == "create" {
			if op.Document != nil {
				docJSON, _ := json.Marshal(op.Document)
				buf.Write(docJSON)
				buf.WriteString("\n")
			}
		} else if op.Operation == "update" {
			updateDoc := map[string]interface{}{}
			if op.Document != nil {
				updateDoc["doc"] = op.Document
			}
			if op.Script != nil {
				updateDoc["script"] = op.Script
			}
			docJSON, _ := json.Marshal(updateDoc)
			buf.Write(docJSON)
			buf.WriteString("\n")
		}
	}

	// Send bulk request using doBulkRequest for NDJSON format
	path := "/_bulk"
	respBody, err := d.client.doBulkRequest(ctx, "POST", path, buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to execute bulk operation: %w", err)
	}

	var result BulkResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse bulk response: %w", err)
	}

	return &result, nil
}
