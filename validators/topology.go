package validators

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"time"

	"github.com/Fantom-foundation/go-opera/gossip"
	"github.com/Fantom-foundation/go-opera/gossip/emitter"
	"github.com/Fantom-foundation/go-opera/validators/service"
	"github.com/Fantom-foundation/go-opera/valkeystore/encryption"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/log"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
)

type TopologyProvider interface {
	// RegisterValidator probably is not necessary in production,
	// but is useful in demo mode so we can register as many validators as requested
	RegisterValidator(idx.ValidatorID, peer.ID, string) error
	GetTopology() (*Topology, error)
}

type Topology struct {
	ListenAddr  map[idx.ValidatorID]string
	Connections map[idx.ValidatorID][]*Validator
}

type Validator struct {
	ID         idx.ValidatorID
	PublicKey  *ecdsa.PublicKey
	ListenAddr string
	City       string
}

func SetupValidatorConnections(
	listenAddr string,
	cfg emitter.ValidatorConfig,
	key *encryption.PrivateKey,
	topologyProvider TopologyProvider,
	gossipSvc *gossip.Service) (service.Validator, error) {
	log.Debug("creating libp2p node...")
	libp2pPrvKey, err := crypto.UnmarshalSecp256k1PrivateKey(key.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to create a libp2p crypto key from the ecdsa key: %w", err)
	}
	libp2pNode, err := libp2p.New(
		libp2p.ListenAddrStrings(listenAddr),
		libp2p.Identity(libp2pPrvKey),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create libp2p node: %w", err)
	}
	thisID := libp2pNode.ID()

	if err := topologyProvider.RegisterValidator(cfg.ID, thisID, listenAddr); err != nil {
		return nil, err
	}

	topology, err := topologyProvider.GetTopology()
	if err != nil {
		return nil, err
	}

	if topology.ListenAddr[cfg.ID] != fmt.Sprintf("%s/ipfs/%s", listenAddr, thisID.String()) {
		return nil, fmt.Errorf("topology provider lists this node's ListenAddr to be %s, but got %s. Can't proceed!", topology.ListenAddr[cfg.ID], listenAddr)
	}

	log.Debug("creating validator service...")
	svc := NewService(libp2pNode, libp2pPrvKey, gossipSvc)

	myconns := topology.Connections[cfg.ID]
	if len(myconns) == 0 {
		return nil, errors.New("no validator connections!")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	for _, conn := range myconns {
		log.Debug("connecting to validator...")
		if err := svc.ConnectToValidator(ctx, conn); err != nil {
			return nil, err
		}
	}
	log.Info("connected to all libp2p validators successfully")

	return svc, nil
}
