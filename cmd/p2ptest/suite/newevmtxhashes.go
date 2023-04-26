package suite

import (
	"bytes"
	"fmt"

	"github.com/Fantom-foundation/go-opera/cmd/p2ptest/proto"
	"github.com/Fantom-foundation/go-opera/gossip"
	"github.com/ethereum/go-ethereum/common"
)

var defaultHashContentSize = 100

func NewEvmTxHashes() proto.Interaction {

	return proto.Interaction{
		Input: proto.Input{
			Msg:  createHashes(defaultHashContentSize),
			Code: gossip.NewEvmTxHashesMsg,
		},
		Output: []proto.Output{
			{
				Msg:  &[]common.Hash{},
				Code: gossip.GetEvmTxsMsg,
				Verify: func(input interface{}, received interface{}) error {
					inputMsg, ok := input.([]common.Hash)
					if !ok {
						return fmt.Errorf("expected input to be of type %T, but got %T", inputMsg, input)
					}
					receivedMsg, ok := received.(*[]common.Hash)
					if !ok {
						return fmt.Errorf("expected output to be of type %T, but got %T", receivedMsg, received)
					}
					if len(inputMsg) != len(*receivedMsg) {
						return fmt.Errorf("expected hash length of %d, got %d", len(inputMsg), len(*receivedMsg))
					}
					for i := 0; i < len(inputMsg); i++ {
						if bytes.Compare(inputMsg[i].Bytes(), (*receivedMsg)[i].Bytes()) != 0 {
							return fmt.Errorf("entry %d of received GetEvmTxs does not match the one in input", i)
						}
					}
					return nil
				},
			},
		},
		Label: "Send NewEvmTxHashes, expect GetEvmTxs",
	}
}
