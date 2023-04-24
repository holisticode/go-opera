package proto

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Fantom-foundation/go-opera/gossip"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

var (
	testProtocolName    = gossip.ProtocolName 
	testProtocolVersion = uint(gossip.ProtocolVersion)
)

type testPeer struct {
	p2pPeer *p2p.Peer
	rw      p2p.MsgReadWriter
}

type sender interface {
	Send(msg interface{}, code uint64) error
}

type receiver interface {
	Receive() (chan []byte, error)
}

const (
	contentType = "application/json"
)

var (
	_ sender = &defaultSender{}
	//_ receiver = &defaultReceiver{}

	ErrSendFailed = errors.New("sending message failed")
)

type defaultSender struct {
	urls []string
	enodes []*enode.Node
	srv    *p2p.Server
	peers  []*testPeer
}

func NewSender(urls []string) (*defaultSender, error) {
	enodes := make([]*enode.Node, len(urls))
	s := &defaultSender{
		enodes: enodes,
		peers:  []*testPeer{},
		urls: urls,
	}
	s.srv = s.startTestServer()

	/*
		listener, err := net.Listen("tcp", "0.0.0.0:30303")
		if err != nil {
			return nil, err
		}
	*/
	return s, nil
}

func (s *defaultSender) createTestProtocol() []p2p.Protocol {
	return []p2p.Protocol{
		{
			Name:    testProtocolName,
			Version: testProtocolVersion,
			Length:  gossip.EPsStreamResponse + 1,
			Run: func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
				return s.run(p, rw)
			},
			/*
				NodeInfo: func() interface{} {
					"test",
				},
				PeerInfo: func(id enode.ID) interface{} {
					return nil
				},
			*/
		},
	}
}

func (s *defaultSender) run(p *p2p.Peer, rw p2p.MsgReadWriter) error {
	fmt.Println("run")
	peer := &testPeer{p2pPeer: p, rw: rw}
	s.peers = append(s.peers, peer)
	for {
		msg, err := rw.ReadMsg()
		if err != nil {
			return err
		}
		defer msg.Discard()

		fmt.Printf("received msg: %d\n", msg.Code)

	}

}

func (s *defaultSender) Send(msg interface{}, code uint64) error {
	/*
	size, reader, err := rlp.EncodeToReader(bmsg)
	if err != nil {
		return err
	}
	msg := p2p.Msg{
		Code:    code,
		Size:    uint32(size),
		Payload: reader,
	}
	*/
	var err error
	for i, u := range s.urls {
		if strings.HasPrefix(u, "enode") {
			s.enodes[i], err = enode.Parse(enode.ValidSchemes, u)
			if err != nil {
				return err
			}
		} else {
			//TODO: call RPC and extract enode
		}
		s.srv.AddPeer(s.enodes[i])
	}

	time.Sleep(5*time.Second)
	fmt.Println(len(s.srv.TrustedNodes))
	for _,p := range s.peers {
		fmt.Println(code)
		fmt.Println(msg)
		p2p.Send(p.rw, code, msg)
	}

	return nil
}

func (s *defaultSender) startTestServer() *p2p.Server {
	logger := log.New(context.Background())
	logger.SetHandler(log.LvlFilterHandler(log.LvlTrace,log.StreamHandler(os.Stderr, log.TerminalFormat(true))))
	config := p2p.Config{
		Name:        "test",
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

/*

type defaultReceiver struct {
	wg       sync.WaitGroup
	listener net.Listener
	msg      chan []byte
}

func (r *defaultReceiver) Receive() (chan []byte, error) {
	return r.msg, nil
}

func NewReceiver() (*defaultReceiver, error) {
	var err error
	r := &defaultReceiver{
		msg: make(chan []byte),
	}
	r.listener, err = net.Listen("tcp", "0.0.0.0:30303")
	if err != nil {
		return nil, err
	}
	// wg.Add(1)
	go r.listenLoop()
	return r, nil
}

func (r *defaultReceiver) listenLoop() {
	for {
		// Listen for an incoming connection
		conn, err := r.listener.Accept()
		if err != nil {
			panic(err)
		}
		//defer close(connected)
		defer srv.Stop()
		// Handle connections in a new goroutine
		go func(conn net.Conn) {
			buf := make([]byte, 1024)
			_, err := conn.Read(buf)
			if err != nil {
				fmt.Printf("Error reading: %#v\n", err)
				return
			}
			r.msg <- buf
			tcpAddr := conn.RemoteAddr().(*net.TCPAddr)

			rlpconn := rlpx.NewConn(conn, &srv.PrivateKey.PublicKey)

			node := enode.NewV4(remid, tcpAddr.IP, tcpAddr.Port, 0)
			srv.AddPeer(node)
			conn.Close()
		}(conn)
	}
}


func TestServerListen(t *testing.T) {
	// start the test server
	//connected := make(chan *Peer)
	srv := startTestServer()
	//defer close(connected)
	defer srv.Stop()

	// dial the test server
	conn, err := net.DialTimeout("tcp", srv.ListenAddr, 5*time.Second)
	if err != nil {
		t.Fatalf("could not dial: %v", err)
	}
	defer conn.Close()

	select {
	case peer := <-connected:
		if peer.LocalAddr().String() != conn.RemoteAddr().String() {
			t.Errorf("peer started with wrong conn: got %v, want %v",
				peer.LocalAddr(), conn.RemoteAddr())
		}
		peers := srv.Peers()
		if !reflect.DeepEqual(peers, []*Peer{peer}) {
			t.Errorf("Peers mismatch: got %v, want %v", peers, []*Peer{peer})
		}
	case <-time.After(1 * time.Second):
		t.Error("server did not accept within one second")
	}
}

*/
