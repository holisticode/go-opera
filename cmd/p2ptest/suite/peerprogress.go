package suite

import (
	"github.com/Fantom-foundation/go-opera/cmd/p2ptest/proto"
	"github.com/Fantom-foundation/go-opera/gossip"
	"github.com/Fantom-foundation/lachesis-base/hash"
)

func PeerProgress() proto.Interaction {

	return proto.Interaction{
		Label: "Send PeerProgress, no response expected",
		Input: proto.Input{
			Msg: gossip.PeerProgress{
				Epoch:            0,
				LastBlockIdx:     1,
				LastBlockAtropos: hash.Event{},
			},
			Code: gossip.ProgressMsg,
		},
		Output: []proto.Output{},
	}
}
