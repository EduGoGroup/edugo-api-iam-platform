package model

import "time"

// LoginAttempt maps to auth.login_attempts table
type LoginAttempt struct {
	ID          int       `gorm:"column:id;primaryKey;autoIncrement"`
	Identifier  string    `gorm:"column:identifier;not null"`
	AttemptType string    `gorm:"column:attempt_type;not null"`
	Successful  bool      `gorm:"column:successful;not null;default:false"`
	UserAgent   *string   `gorm:"column:user_agent"`
	IPAddress   *string   `gorm:"column:ip_address"`
	AttemptedAt time.Time `gorm:"column:attempted_at;not null;default:now()"`
}

func (LoginAttempt) TableName() string {
	return "auth.login_attempts"
}
