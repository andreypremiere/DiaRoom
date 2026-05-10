package responses

type Content struct {
	Folders []*FolderShow  `json:"folders"`
	Items   []*ItemShow `json:"items"`
}