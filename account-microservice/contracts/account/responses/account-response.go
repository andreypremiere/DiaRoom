package responses

type AuthResponse struct {
	AccessToken string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	IsConfigured bool `json:"isConfigured"`
}

type RefreshTokens struct {
	AccessToken string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}