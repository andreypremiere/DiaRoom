package requests

type UpdatingCategoriesRequest struct {
	Categories []string `json:"categories"`
}