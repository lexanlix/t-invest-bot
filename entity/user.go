package entity

import (
	"github.com/russianinvestments/invest-api-go-sdk/investgo"
)

const (
	StateStart = iota
	StateUserCreated
	StateGotInvestToken
	StateSelectedAccount
)

type User struct {
	ChatId   int64
	Username string
	State    int
	Invest   Invest
}

type Invest struct {
	investgo.Config

	AccountNameById    map[string]string
	TracingAccountName string
}
