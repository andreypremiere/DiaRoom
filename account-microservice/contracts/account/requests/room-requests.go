package requests

type UpdateRoomRequest struct {
	RoomUniqueID      string   `json:"roomUniqueId"`
	RoomName          string   `json:"roomName"`
	Bio               string   `json:"bio"`
	Categories        []string `json:"categories"`
	AvatarFileName    string   `json:"avatar_filename"`
	BackgroundFileName string   `json:"background_filename"`
}