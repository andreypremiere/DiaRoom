package models

type PostDraft struct {
	Title        string      `json:"title"`
	CategorySlug string      `json:"categorySlug"`
	Hashtags     []string    `json:"hashtags"`
	Blocks       []PostBlock `json:"blocks"` 
}

type PostBlock interface {
	GetBlockType() string
}

// BaseBlock содержит общие поля для всех блоков
type BaseBlock struct {
	BlockType string `json:"blockType"`
}

// TextBlockPost соответствует TextBlockPost во Flutter
type TextBlockPost struct {
	BaseBlock
	Value    string `json:"value"`
	TextType string `json:"textType"`
}

func (b TextBlockPost) GetBlockType() string { return b.BlockType }


// PhotoBlockPost соответствует PhotoBlockPost во Flutter
type PhotoBlockPost struct {
	BaseBlock
	LocalPaths      []string `json:"localPaths"`
	PublicUrls      []string `json:"publicUrls"`
	PresignedUrls   []string `json:"presignedUrls"`
	MethodViewPhoto string   `json:"methodViewPhoto"`
	PhotoSizes      []int    `json:"photoSizes"`
}

func (b PhotoBlockPost) GetBlockType() string { return b.BlockType }

// VideoBlockPost соответствует VideoBlockPost во Flutter
type VideoBlockPost struct {
	BaseBlock
	LocalPath           string `json:"localPath"`
	PublicUrl           string `json:"publicUrl"`
	PresignedUrl        string `json:"presignedUrl"`
	PreviewLocalPath    string `json:"previewLocalPath"`
	PreviewPublicUrl    string `json:"previewPublicUrl"`
	PreviewPresignedUrl string `json:"previewPresignedUrl"`
	FileSize            int64  `json:"fileSize"`
	DurationMs          int64  `json:"durationMs"` 
}

func (b VideoBlockPost) GetBlockType() string { return b.BlockType }