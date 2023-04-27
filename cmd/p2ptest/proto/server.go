package proto

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/Fantom-foundation/go-opera/gossip"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"go.uber.org/zap"
)

var (
	// the following protocol identifiers need to be the same
	// as our analyzed protocol or peers will reject
	testProtocolName    = gossip.ProtocolName
	testProtocolVersion = uint(gossip.ProtocolVersion)
)

type testPeer struct {
	p2pPeer *p2p.Peer
	rw      p2p.MsgReadWriter
}

type p2pProtocolTest interface {
	Send(msg interface{}, code uint64) error
	Receive() chan p2p.Msg
	Initalized() chan struct{}
}

const (
	contentType = "application/json"
)

var (
	_ p2pProtocolTest = &defaultProtocol{}

	ErrSendFailed = errors.New("sending message failed")
)

type defaultProtocol struct {
	urls         []string
	enodes       []*enode.Node
	srv          *p2p.Server
	peers        []*testPeer
	receiveCh    chan p2p.Msg
	logger       *zap.Logger
	protoRunning chan struct{}
}

func New(urls []string, logger *zap.Logger) (*defaultProtocol, error) {
	enodes := make([]*enode.Node, len(urls))
	s := &defaultProtocol{
		enodes:       enodes,
		peers:        []*testPeer{},
		urls:         urls,
		receiveCh:    make(chan p2p.Msg),
		protoRunning: make(chan struct{}),
		logger:       logger,
	}
	s.srv = s.startTestServer()
	var err error
	for i, u := range s.urls {
		if strings.HasPrefix(u, "enode") {
			s.enodes[i], err = enode.Parse(enode.ValidSchemes, u)
			if err != nil {
				return nil, err
			}
		} else {
			//TODO: call RPC and extract enode
		}
		s.srv.AddPeer(s.enodes[i])
	}

	return s, nil
}

func (s *defaultProtocol) createTestProtocol() []p2p.Protocol {
	return []p2p.Protocol{
		{
			Name:    testProtocolName,
			Version: testProtocolVersion,
			Length:  gossip.EPsStreamResponse + 1,
			Run: func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
				return s.run(p, rw)
			},
		},
	}
}

func (s *defaultProtocol) run(p *p2p.Peer, rw p2p.MsgReadWriter) error {
	s.logger.Debug("running protocol for peer", zap.String("peer", p.String()))
	s.protoRunning <- struct{}{}
	peer := &testPeer{p2pPeer: p, rw: rw}
	s.peers = append(s.peers, peer)
	for {
		msg, err := rw.ReadMsg()
		if err != nil {
			return err
		}
		defer msg.Discard()
		s.logger.Debug("received message", zap.Uint64("code", msg.Code))

		// TODO this doesn't work for multiple peers
		// they'd use the same channel and thus interfere in the sequence
		s.receiveCh <- msg

	}

}

func (s *defaultProtocol) Send(msg interface{}, code uint64) error {
	fmt.Println(len(s.peers))
	for _, p := range s.peers {
		s.logger.Debug("sending message to peer", zap.Uint64("code", code), zap.String("peer", p.p2pPeer.String()))
		p2p.Send(p.rw, code, msg)
	}

	return nil
}

func (s *defaultProtocol) Receive() chan p2p.Msg {
	return s.receiveCh
}

func (s *defaultProtocol) Initalized() chan struct{} {
	return s.protoRunning
}
func (s *defaultProtocol) startTestServer() *p2p.Server {
	logger := log.New(context.Background())
	//logger.SetHandler(log.LvlFilterHandler(log.LvlTrace, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))
	logger.SetHandler(log.LvlFilterHandler(log.LvlError, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))
	config := p2p.Config{
		Name:        "testRunner",
		MaxPeers:    10,
		ListenAddr:  "127.0.0.1:0",
		NoDiscovery: true,
		PrivateKey:  newkey(),
		Logger:      logger,
	}
	server := &p2p.Server{
		Config: config,
	}
	server.Protocols = s.createTestProtocol()
	if err := server.Start(); err != nil {
		panic(fmt.Sprintf("Could not start server: %v", err))
	}
	return server
}

func newkey() *ecdsa.PrivateKey {
	key, err := crypto.GenerateKey()
	if err != nil {
		panic("couldn't generate key: " + err.Error())
	}
	return key
}
