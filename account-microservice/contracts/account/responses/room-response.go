package responses

type RoomResponse struct {
	RoomUniqueId string `json:"roomUniqueId"`
	RoomName string `json:"roomName"`
	ListCategory []string `json:"listCategory"`
	Bio string `json:"bio"`
	AvatarPath string `json:"avatarPath"`
	BackgroundPath string `json:"backgroundPath"`
}

type UpdateRoomResponse struct {
	PresignedUrlAvatar string `json:"presignedUrlAvatar"`
	PresignedUrlBackground string `json:"presignedUrlBackground"`
}

type RoomInfo struct {
    AvatarUrl string `json:"avatarUrl"`
    RoomName  string `json:"roomName"`
}