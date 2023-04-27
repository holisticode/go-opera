package suite

import (
	"bytes"
	"fmt"

	"github.com/Fantom-foundation/go-opera/cmd/p2ptest/proto"
	"github.com/Fantom-foundation/go-opera/gossip"
	"github.com/ethereum/go-ethereum/common"
)

func Handshake() proto.Interaction {

	return proto.Interaction{
		Label: "Send Handshake, expect one HandshakeMsg and one PeerProgress",
		Input: proto.Input{
			Msg: gossip.HandshakeData{
				ProtocolVersion: 63,
				NetworkID:       0,
				// this hash is extracted from the demo
				// TODO: should be configurable
				Genesis: common.HexToHash("0x2c210befc091e71047cc7efb2b7789805c9dbd3081f08e67ecc9ca2236a510c0"),
			},
			Code: gossip.HandshakeMsg,
		},
		Output: []proto.Output{
			{
				Code: 0,
				Msg:  gossip.HandshakeData{},
				Verify: func(input interface{}, received interface{}) error {
					inputMsg, ok := input.(gossip.HandshakeData)
					if !ok {
						return fmt.Errorf("expected input to be of type %T, but got %T", inputMsg, input)
					}
					receivedMsg, ok := received.(gossip.HandshakeData)
					if !ok {
						return fmt.Errorf("expected output to be of type %T, but got %T", receivedMsg, received)
					}

					if inputMsg.ProtocolVersion != receivedMsg.ProtocolVersion {
						return fmt.Errorf("protocol versions don't match, mine: %d, received: %d", inputMsg.ProtocolVersion, receivedMsg.ProtocolVersion)
					}
					if inputMsg.NetworkID != receivedMsg.NetworkID {
						return fmt.Errorf("networkIDs don't match, mine: %d, received: %d", inputMsg.NetworkID, receivedMsg.NetworkID)
					}
					if bytes.Compare(inputMsg.Genesis.Bytes(), receivedMsg.Genesis.Bytes()) != 0 {
						return fmt.Errorf("genesis don't match, mine: %s, received: %s", inputMsg.Genesis.Hex(), receivedMsg.Genesis.Hex())
					}
					return nil

				},
			},
			{
				Code: 1,
				Msg:  gossip.PeerProgress{},
				Verify: func(input interface{}, received interface{}) error {
					receivedMsg, ok := received.(gossip.PeerProgress)
					if !ok {
						return fmt.Errorf("expected output to be of type %T, but got %T", receivedMsg, received)
					}
					// We expect the peer to be in the future...
					if receivedMsg.Epoch < 1 {
						return fmt.Errorf("unexpected epoch: %d", receivedMsg.Epoch)
					}
					/*
						if receivedMsg.LastBlockAtropos.IsZero() {
							return fmt.Errorf("unexpected zero atropos")
						}
					*/
					if receivedMsg.LastBlockIdx < 1 {
						return fmt.Errorf("unexpected last block index: %d", receivedMsg.LastBlockIdx)
					}
					return nil
				},
			},
		},
	}
}
