package value_object

import (
	"errors"
	"golang.org/x/crypto/bcrypt"
)

// Password 密码值对象
type Password struct {
	hash string
}

// NewPassword 创建密码值对象
func NewPassword(plainPassword string) (Password, error) {
	if len(plainPassword) < 6 {
		return Password{}, errors.New("password must be at least 6 characters")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.DefaultCost)
	if err != nil {
		return Password{}, err
	}

	return Password{hash: string(hash)}, nil
}

// NewPasswordFromHash 从哈希值创建密码（用于从数据库恢复）
func NewPasswordFromHash(hash string) Password {
	return Password{hash: hash}
}

// Verify 验证密码
func (p Password) Verify(plainPassword string) bool {
	return bcrypt.CompareHashAndPassword([]byte(p.hash), []byte(plainPassword)) == nil
}

// Hash 返回密码哈希值
func (p Password) Hash() string {
	return p.hash
}
