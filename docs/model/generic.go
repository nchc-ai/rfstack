package docs

type GenericErrorResponse struct {
	Error   bool   `json:"error" example:"true" format:"bool"`
	Message string `json:"message" example:"error response message" format:"string"`
}

type GenericOKResponse struct {
	Error   bool   `json:"error" example:"false" format:"bool"`
	Message string `json:"message" example:"response message" format:"string"`
}

type OauthUser struct {
	User     string `json:"user" example:"shan-test@teacher"`
}
