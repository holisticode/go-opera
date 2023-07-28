package service

import (
	"context"

	"github.com/Fantom-foundation/go-opera/inter"
	"github.com/ethereum/go-ethereum/core/types"
)

type Validator interface {
	ForwardEvents(ctx context.Context, events inter.EventPayloads)
	ForwardTxs(ctx context.Context, events types.Transactions)
	ConnectToValidator(ctx context.Context, validator interface{}) error
}
