package validators

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	mrand "math/rand"
	"sync"
	"time"

	"github.com/Fantom-foundation/go-opera/gossip"
	"github.com/Fantom-foundation/go-opera/inter"
	"github.com/Fantom-foundation/go-opera/validators/service"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	pool "github.com/libp2p/go-buffer-pool"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	peerstore "github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
)

const (
	AliveSize       = 32
	TxsServiceID    = "/fantom/validator/txs/1.0.0"
	EventsServiceID = "/fantom/validator/evt/1.0.0"
	AliveServiceID  = "/fantom/validator/alive/1.0.0"
	ServiceName     = "fantom.validator.service"

	connTimeout = 10 * time.Second
)

var _ service.Validator = &libp2pValidatorService{}

type libp2pValidatorService struct {
	gossipSvc *gossip.Service
	key       crypto.PrivKey
	Host      host.Host
	lock      sync.Mutex
}

func NewService(h host.Host, key crypto.PrivKey, gossipSvc *gossip.Service) service.Validator {
	vs := &libp2pValidatorService{
		Host:      h,
		key:       key,
		gossipSvc: gossipSvc,
	}
	h.SetStreamHandler(TxsServiceID, vs.ValidatorTxsServiceHandler)
	h.SetStreamHandler(EventsServiceID, vs.ValidatorEventsServiceHandler)
	h.SetStreamHandler(AliveServiceID, vs.ValidatorAliveServiceHandler)
	return vs
}

func aliveError(err error) chan error {
	ch := make(chan error, 1)
	ch <- err
	close(ch)
	return ch
}

func (v *libp2pValidatorService) ForwardTxs(ctx context.Context, txs types.Transactions) {
	txBytes, err := rlp.EncodeToBytes(&txs)
	if err != nil {
		log.Warn("failed to encode data", "error", err)
		return
	}
	v.executeForward(ctx, txBytes, TxsServiceID)
}

func (v *libp2pValidatorService) ForwardEvents(ctx context.Context, events inter.EventPayloads) {
	evtBytes, err := rlp.EncodeToBytes(&events)
	if err != nil {
		log.Warn("failed to encode data", "error", err)
		return
	}
	v.executeForward(ctx, evtBytes, EventsServiceID)
}

func (v *libp2pValidatorService) executeForward(ctx context.Context, sendBytes []byte, protoID protocol.ID) {
	for _, p := range v.Host.Peerstore().Peers() {
		if p == v.Host.ID() {
			// dont' forward to self
			continue
		}
		s, err := v.Host.NewStream(network.WithUseTransient(ctx, "validator"), p, protoID)
		if err != nil {
			// NOTE: We are doing best-effort here: anything related to libp2p validators for now
			// which creates an error is ignored but logged. In that case we rely on the normal
			// devp2p propagation, in order to not introduce a new critical path in case of failure
			log.Warn("failed to create a stream for peer", "peer", p.String(), "error", err)
			return
		}

		if err := s.Scope().SetService(ServiceName); err != nil {
			log.Warn("error attaching stream to validator service", "error", err)
			s.Reset()
			return
		}

		lenSendBytes := len(sendBytes)
		// TODO verify reserve reqs
		if err := s.Scope().ReserveMemory(lenSendBytes, network.ReservationPriorityAlways); err != nil {
			log.Warn("error reserving memory for validator stream", "error", err)
			s.Reset()
			return
		}

		buf := pool.Get(lenSendBytes)

		log.Trace("reading message into buffer...")
		copy(buf, sendBytes)

		release := func() {
			s.Scope().ReleaseMemory(lenSendBytes)
			pool.Put(buf)
		}

		log.Trace("sending message to wire...")
		if _, err := s.Write(buf); err != nil {
			log.Warn("failed to send message to peer", "peer", p.String(), "error", err)
			release()
			return
		}
		log.Debug("message successfully sent")
		release()
	}
}

func (v *libp2pValidatorService) ConnectToValidator(ctx context.Context, validator interface{}) error {
	var (
		conn *Validator
		ok   bool
	)

	if conn, ok = validator.(*Validator); !ok {
		return fmt.Errorf("expected Validator type, but got %T", validator)
	}

	addr, err := v.decrypt(conn.ListenAddr)
	if err != nil {
		return err
	}
	log.Debug("connecting to validator", "address", addr)
	maddr, err := multiaddr.NewMultiaddr(addr)
	if err != nil {
		return err
	}
	peer, err := peerstore.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return err
	}

	retry := true
	for retry {
		select {
		case <-ctx.Done():
			return errors.New("timed out attempting to connect to validator!")
		case <-time.After(500 * time.Millisecond):
			if err := v.Host.Connect(ctx, *peer); err != nil {
				return err
			}
			retry = false
		}
	}
	log.Debug("connection established")
	/*
		ch := v.PingValidator(ctx, peer.ID)
		for i := 0; i < 5; i++ {
			res := <-ch
			fmt.Println("pinged", addr, "err", res)
		}
	*/
	return nil
}

func (v *libp2pValidatorService) baseServiceHandler(s network.Stream) []byte {
	v.lock.Lock()
	defer v.lock.Unlock()
	if err := s.Scope().SetService(ServiceName); err != nil {
		log.Warn("error attaching stream to validator service", "error", err)
		s.Reset()
		return nil
	}

	if err := s.Scope().ReserveMemory(inter.ProtocolMaxMsgSize, network.ReservationPriorityAlways); err != nil {
		log.Warn("error reserving memory for validator stream", "error", err)
		s.Reset()
		return nil
	}
	defer s.Scope().ReleaseMemory(inter.ProtocolMaxMsgSize)

	buf := pool.Get(inter.ProtocolMaxMsgSize)
	defer pool.Put(buf)

	log.Trace("ValidatorEventsServiceHandler reading into buffer...")
	_, err := s.Read(buf)
	//_, err := io.ReadFull(s, buf)
	if err != nil {
		log.Error("failed reading received message", "error", err)
		return nil
	}
	log.Trace("ValidatorEventsServiceHandler reading into buffer done")

	result := make([]byte, len(buf))
	copy(result, buf)
	return result
}

func (v *libp2pValidatorService) ValidatorEventsServiceHandler(s network.Stream) {
	var events inter.EventPayloads

	log.Trace("ValidatorEventsServiceHandler running")
	buf := v.baseServiceHandler(s)
	reader := bytes.NewReader(buf)
	if err := rlp.Decode(reader, &events); err != nil {
		log.Error("failed decoding message", "error", err)
		return
	}
	log.Trace("ValidatorEventsServiceHandler passing event to gossip service")
	v.gossipSvc.HandleValidatorEvents(events)
}

func (v *libp2pValidatorService) ValidatorTxsServiceHandler(s network.Stream) {
	var txs types.Transactions

	buf := v.baseServiceHandler(s)
	reader := bytes.NewReader(buf)
	if err := rlp.Decode(reader, &txs); err != nil {
		log.Error("failed decoding message", "error", err)
		return
	}
	v.gossipSvc.HandleValidatorTxs(txs)
}

func (v *libp2pValidatorService) ValidatorAliveServiceHandler(s network.Stream) {
	if err := s.Scope().SetService(ServiceName); err != nil {
		log.Warn("error attaching stream to alive service", "error", err)
		s.Reset()
		return
	}

	if err := s.Scope().ReserveMemory(AliveSize, network.ReservationPriorityAlways); err != nil {
		log.Warn("error reserving memory for alive stream", "error", err)
		s.Reset()
		return
	}
	defer s.Scope().ReleaseMemory(AliveSize)

	buf := pool.Get(AliveSize)
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
		fmt.Println("#####")
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

func (v *libp2pValidatorService) AliveValidator(ctx context.Context, peer peer.ID) <-chan error {
	s, err := v.Host.NewStream(network.WithUseTransient(ctx, "validator"), peer, AliveServiceID)
	if err != nil {
		return aliveError(err)
	}

	if err := s.Scope().SetService(ServiceName); err != nil {
		log.Warn("error attaching stream to alive service", "error", err)
		s.Reset()
		return aliveError(err)
	}

	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		log.Error("failed to get cryptographic random", "error", err)
		s.Reset()
		return aliveError(err)
	}
	ra := mrand.New(mrand.NewSource(int64(binary.BigEndian.Uint64(b))))

	ctx, cancel := context.WithCancel(ctx)

	out := make(chan error)
	go func() {
		defer close(out)
		defer cancel()

		for ctx.Err() == nil {
			rtt, err := ping(s, ra)

			// canceled, ignore everything.
			if ctx.Err() != nil {
				return
			}

			// No error, record the RTT.
			if err == nil {
				v.Host.Peerstore().RecordLatency(peer, rtt)
			}

			select {
			case out <- err:
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() {
		// forces the ping to abort.
		<-ctx.Done()
		s.Reset()
	}()

	return out
}

func ping(s network.Stream, randReader io.Reader) (time.Duration, error) {
	if err := s.Scope().ReserveMemory(2*AliveSize, network.ReservationPriorityAlways); err != nil {
		log.Debug("error reserving memory for ping stream", "error", err)
		s.Reset()
		return 0, err
	}
	defer s.Scope().ReleaseMemory(2 * AliveSize)

	buf := pool.Get(AliveSize)
	defer pool.Put(buf)

	if _, err := io.ReadFull(randReader, buf); err != nil {
		return 0, err
	}

	before := time.Now()
	if _, err := s.Write(buf); err != nil {
		return 0, err
	}

	rbuf := pool.Get(AliveSize)
	defer pool.Put(rbuf)

	if _, err := io.ReadFull(s, rbuf); err != nil {
		return 0, err
	}

	if !bytes.Equal(buf, rbuf) {
		return 0, errors.New("ping packet was incorrect")
	}

	return time.Since(before), nil
}

func (v *libp2pValidatorService) decrypt(addr string) (string, error) {
	//return crypto.Decrypt(addr, v.key)
	return addr, nil
}
