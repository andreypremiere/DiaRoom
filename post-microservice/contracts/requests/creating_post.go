package requests

import "encoding/json"

type PostDraftRequest struct {
	Title        string            `json:"title"`
	CategorySlug string            `json:"categorySlug"`
	Hashtags     []string          `json:"hashtags"`
	Blocks       []json.RawMessage `json:"blocks"` // "Сырые" байты каждого блока
}