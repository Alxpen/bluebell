package models

type User struct {
	UserID int16 `db:"user_id"`
	Username string `db:"username"`
	Password string `db:"password"`
	Token string
}