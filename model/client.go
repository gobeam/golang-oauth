package model

// Clients is model for oauth clients
type Clients struct {
	Model
	UserId   int64  `db:"user_id"`
	Name     string `db:"name"`
	Secret   string `db:"secret"`
	Revoked  bool   `db:"revoked"`
}
