package models

import "encoding/json"

type SaveCanvasRequest struct {
	Payload json.RawMessage `json:"payload"`
}