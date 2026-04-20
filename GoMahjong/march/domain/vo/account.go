package vo

import (
	"errors"
	"strings"
)

type Account string

func NewAccount(account string) (Account, error) {
	account = strings.TrimSpace(account)
	if account == "" {
		return "", errors.New("account cannot be empty")
	}
	if len(account) < 3 {
		return "", errors.New("account must be at least 3 characters")
	}
	return Account(account), nil
}

func NewAccountFromString(account string) Account {
	return Account(account)
}

func (a Account) String() string {
	return string(a)
}
