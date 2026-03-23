package models

type PreviewRequest struct {
	PreviewId string `json:"previewId"`
	PathPreview string `json:"pathPreview,omitempty"`
}

type PreviewResponse struct {
	PreviewReq PreviewRequest 
	PresignedURL string `json:"presignedUrl"`
	PublicURL    string `json:"publicUrl"`
}