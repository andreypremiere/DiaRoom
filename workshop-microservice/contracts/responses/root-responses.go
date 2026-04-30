package responses

type Root struct {
	Folders []*FolderShow `json:"folders"`
}