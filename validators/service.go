package validators

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"io"
	"time"

	"github.com/ethereum/go-ethereum/log"
	pool "github.com/libp2p/go-buffer-pool"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	peerstore "github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

const (
	PingSize    = 32
	ID          = "/fantom/validator/1.0.0"
	ServiceName = "fantom.validator.service"

	connTimeout = 10 * time.Second
)

type ValidatorService struct {
	key  *ecdsa.PrivateKey
	Host host.Host
}

func NewValidatorService(h host.Host, key *ecdsa.PrivateKey) *ValidatorService {
	vs := &ValidatorService{
		Host: h,
		key:  key,
	}
	h.SetStreamHandler(ID, vs.ValidatorServiceHandler)
	return vs
}

func (v *ValidatorService) ConnectToValidator(conn *Validator) error {
	addr, err := v.decrypt(conn.ListenAddr)
	if err != nil {
		return err
	}
	fmt.Println(addr)
	maddr, err := multiaddr.NewMultiaddr(addr)
	if err != nil {
		return err
	}
	peer, err := peerstore.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return v.Host.Connect(ctx, *peer)
}

func (v *ValidatorService) ValidatorServiceHandler(s network.Stream) {
	if err := s.Scope().SetService(ServiceName); err != nil {
		log.Debug("error attaching stream to ping service: %s", err)
		s.Reset()
		return
	}

	if err := s.Scope().ReserveMemory(PingSize, network.ReservationPriorityAlways); err != nil {
		log.Debug("error reserving memory for ping stream: %s", err)
		s.Reset()
		return
	}
	defer s.Scope().ReleaseMemory(PingSize)

	buf := pool.Get(PingSize)
	defer pool.Put(buf)

	errCh := make(chan error, 1)
	defer close(errCh)
	timer := time.NewTimer(connTimeout)
	defer timer.Stop()

	go func() {
		select {
		case <-timer.C:
			log.Debug("ping timeout")
		case err, ok := <-errCh:
			if ok {
				log.Debug(err.Error())
			} else {
				log.Error("ping loop failed without error")
			}
		}
		s.Close()
	}()

	for {
		_, err := io.ReadFull(s, buf)
		if err != nil {
			errCh <- err
			return
		}

		_, err = s.Write(buf)
		if err != nil {
			errCh <- err
			return
		}

		timer.Reset(connTimeout)
	}
}

func (v *ValidatorService) decrypt(addr string) (string, error) {
	//return crypto.Decrypt(addr, v.key)
	return addr, nil
}
