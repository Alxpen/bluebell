package models

type ParamSignUp struct {
	Username string `json:"username"`
	Password string `json:"password"`
	RePassword string `json:"re_password"`
}