package mock

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Fantom-foundation/go-opera/validators"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/log"
	"github.com/libp2p/go-libp2p/core/peer"
)

const (
	// TODO configurable?
	defaultPort = 9669
)

var (
	_ validators.TopologyProvider = &mockTopologyProvider{}
)

type mockTopologyProvider struct {
}

func NewMockTopologyProvider() validators.TopologyProvider {
	return &mockTopologyProvider{}
}

func (p *mockTopologyProvider) RegisterValidator(id idx.ValidatorID, libp2pID peer.ID, listenAddr string) error {
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/setListenAddrForValidator?id=%d&listen-addr=%s/ipfs/%s",
		defaultPort,
		id,
		listenAddr,
		libp2pID.String()))
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

func (p *mockTopologyProvider) GetTopology() (*validators.Topology, error) {
	timeout := time.NewTimer(20 * time.Second)
	ticker := time.NewTicker(1 * time.Second)
	loop := true
	for loop {
		select {
		case <-timeout.C:
			return nil, errors.New("timed out waiting for mock server to be ready")
		case <-ticker.C:
			readyResp, err := http.Get(fmt.Sprintf("http://localhost:%d/ready", defaultPort))
			if err != nil {
				log.Warn("error requesting ready state from mock server: %s", err)
				continue
			}
			if readyResp.StatusCode != http.StatusOK {
				log.Debug("requesting ready state from mock server returned NOT ready")
			} else {
				loop = false
			}
		}
	}

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/getTopology", defaultPort))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	resBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("client: could not read response body: %w\n", err)
	}

	var topology *validators.Topology
	if err := json.Unmarshal(resBody, &topology); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body: %w", err)
	}
	return topology, nil
	/*
		return &Topology{
			Connections: map[idx.ValidatorID][]*Validator{
				1: {
					{
						ID:         idx.ValidatorID(2),
						PublicKey:  nil,
						ListenAddr: "/ip4/127.0.0.1/tcp/9001",
						City:       "Sydney",
					},
					{
						ID:         idx.ValidatorID(3),
						PublicKey:  nil,
						ListenAddr: "/ip4/127.0.0.1/tcp/9002",
						City:       "Sydney",
					},
				},
				2: {
					{
						ID:         idx.ValidatorID(1),
						PublicKey:  nil,
						ListenAddr: "/ip4/127.0.0.1/tcp/9000",
						City:       "Sydney",
					},
					{
						ID:         idx.ValidatorID(3),
						PublicKey:  nil,
						ListenAddr: "/ip4/127.0.0.1/tcp/9002",
						City:       "Sydney",
					},
				},
				3: {
					{
						ID:         idx.ValidatorID(1),
						PublicKey:  nil,
						ListenAddr: "/ip4/127.0.0.1/tcp/9000",
						City:       "Sydney",
					},
					{
						ID:         idx.ValidatorID(2),
						PublicKey:  nil,
						ListenAddr: "/ip4/127.0.0.1/tcp/9001",
						City:       "Sydney",
					},
				},
			},
			ListenAddr: map[idx.ValidatorID]string{
				1: "/ip4/127.0.0.1/tcp/9000",
				2: "/ip4/127.0.0.1/tcp/9001",
				3: "/ip4/127.0.0.1/tcp/9002",
			},
		}
	*/
}
