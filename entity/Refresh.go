package entity

type RefreshResponse struct {
	Message string `json:"message"`
	Result  struct {
		AccessToken string `json:"accessToken"`
	} `json:"result"`
	Status int `json:"status"`
}
