package suite

import "github.com/Fantom-foundation/go-opera/cmd/p2ptest/proto"

func InitialSuite() *proto.Sequence {
	s := &proto.Sequence{}

	s.Steps = append(s.Steps, Handshake())
	s.Steps = append(s.Steps, PeerProgress())
	s.Steps = append(s.Steps, NewEvmTxHashes())

	return s
}
