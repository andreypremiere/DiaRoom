package responses

import "workshop-microservice/models"

type Content struct {
	Folders []*FolderShow  `json:"folders"`
	Items   []*models.Item `json:"items"`
}