package requests

type UpdatingMessage struct {
	MessageID string `json:"messageId"`
	Status    string `json:"status"`
}