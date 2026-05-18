package responses

import "diary-microservice/models"

type GettingMessages struct {
	Messages []*MessageResponse `json:"messages"`
}

type MessageResponse struct {
	*models.Message
	Attachment []*models.Attachment `json:"attachments"`
	Tags []*models.Tag `json:"tags"`
}