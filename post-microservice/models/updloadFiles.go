package models

type PresignedRequest struct {
	Files []UploadFile `json:"files"`
	PostId string `json:"postId"`
	Preview PreviewRequest `json:"previewRequest"`
}

type UploadFile struct {
	UploadID    string `json:"uploadId"`    // uuid с клиента
	Filename    string `json:"filename"`
	ContentType string `json:"contentType"`
	Size        int64  `json:"size,omitempty"`
}

type PresignedResponse struct {
	Files []PresignedFile `json:"files"`
	Preview PreviewResponse `json:"previewResponse"`
}

type PresignedFile struct {
	UploadID     string `json:"uploadId"`
	PresignedURL string `json:"presignedUrl"`
	PublicURL    string `json:"publicUrl"`
}