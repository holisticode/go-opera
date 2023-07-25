package mock

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Fantom-foundation/go-opera/validators"
)

const (
	defaultPort = 9669
)

var DefaultTopologyProvider = MockTopologyProvider

func MockTopologyProvider() (*validators.Topology, error) {
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
				fmt.Println(fmt.Sprintf("error requesting ready state from mock server: %s", err))
				continue
			}
			if readyResp.StatusCode != http.StatusOK {
				fmt.Println("requesting ready state from mock server returned NOT ready")
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
