package value_object

import (
	"errors"
	"strings"
)

// Account 账号值对象
type Account struct {
	value string
}

// NewAccount 创建账号值对象
func NewAccount(account string) (Account, error) {
	account = strings.TrimSpace(account)
	if account == "" {
		return Account{}, errors.New("account cannot be empty")
	}
	if len(account) < 3 {
		return Account{}, errors.New("account must be at least 3 characters")
	}
	return Account{value: account}, nil
}

// NewAccountFromString 从字符串创建账号（用于从数据库恢复）
func NewAccountFromString(account string) Account {
	return Account{value: account}
}

// String 返回账号字符串
func (a Account) String() string {
	return a.value
}

// Equals 比较两个账号是否相等
func (a Account) Equals(other Account) bool {
	return a.value == other.value
}
