package responses

type UpdatingAvatarResponse struct {
	UploadURL string `json:"uploadUrl"` 
	PublicURL string `json:"publicUrl"` 
}