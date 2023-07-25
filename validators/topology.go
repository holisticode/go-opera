package validators

import (
	"crypto/ecdsa"
	"fmt"
	"io"
	"net/http"

	"github.com/Fantom-foundation/go-opera/gossip/emitter"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/peer"
)

type TopologyProvider func() (*Topology, error)

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

func SetupValidatorConnections(listenAddr string, cfg emitter.ValidatorConfig, key *ecdsa.PrivateKey, getTopology TopologyProvider) error {

	fmt.Println("creating libp2p node...")
	/*
		libp2pPrvKey, _, err := crypto.ECDSAKeyPairFromKey(key)
		if err != nil {
			return fmt.Errorf("failed to create a libp2p crypto key from the ecdsa key: %w", err)
		}
	*/
	libp2pNode, err := libp2p.New(
		libp2p.ListenAddrStrings(listenAddr),
	//	libp2p.Identity(libp2pPrvKey),
	)
	if err != nil {
		return fmt.Errorf("failed to create libp2p node: %w", err)
	}
	thisID := libp2pNode.ID()
	fmt.Println(thisID)
	//topology.ListenAddr[cfg.ID] = fmt.Sprintf("%s/%s", topology.ListenAddr[cfg.ID], thisID)
	if err := setTopology(cfg.ID, thisID, listenAddr); err != nil {
		return err
	}

	topology, err := getTopology()
	if err != nil {
		return err
	}
	fmt.Println(topology)

	if topology.ListenAddr[cfg.ID] != fmt.Sprintf("%s/ipfs/%s", listenAddr, thisID.String()) {
		return fmt.Errorf("topology provider lists this node's ListenAddr to be %s, but got %s. Can't proceed!", topology.ListenAddr[cfg.ID], listenAddr)
	}

	fmt.Println("creating validator service...")
	svc := NewValidatorService(libp2pNode, key)

	myconns := topology.Connections[cfg.ID]
	for _, conn := range myconns {
		fmt.Println("connecting to validator...")
		if err := svc.ConnectToValidator(conn); err != nil {
			return err
		}
	}
	fmt.Println("connected to all peers successfully")

	return nil
}

func setTopology(validator idx.ValidatorID, id peer.ID, listenAddr string) error {
	resp, err := http.Get(fmt.Sprintf("http://localhost:9669/setListenAddrForValidator?id=%d&listen-addr=%s/ipfs/%s",
		validator,
		listenAddr,
		id.String()))
	if err != nil {
		return err
	}
	res, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("expected status code %d when setting id on topology, but got %d: %s",
			http.StatusCreated,
			resp.StatusCode,
			string(res))
	}
	return nil
}
