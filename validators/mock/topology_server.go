package mock

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"github.com/Fantom-foundation/go-opera/validators"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/log"
)

type MockTopologyServer struct {
	topology validators.Topology
	port     int
	isReady  bool
	lock     sync.Mutex
}

func NewMockTopologyServer() {
	s := MockTopologyServer{
		port: defaultPort,
		topology: validators.Topology{
			Connections: map[idx.ValidatorID][]*validators.Validator{},
			ListenAddr:  map[idx.ValidatorID]string{},
		},
	}

	http.HandleFunc("/", status)
	http.HandleFunc("/ready", s.ready)
	http.HandleFunc("/setready", s.setReady)
	http.HandleFunc("/getTopology", s.getTopology)
	http.HandleFunc("/getNodesNum", s.getNodesNum)
	http.HandleFunc("/getValidatorsForID", s.getValidatorsForID)
	http.HandleFunc("/setListenAddrForValidator", s.setListenAddrForValidator)
	http.HandleFunc("/getListenAddrForValidator", s.getListenAddrForValidator)

	if err := s.start(); err != nil {
		panic(err)
	}
}

func (s *MockTopologyServer) start() error {
	log.Debug("starting mock topology server...")
	if err := http.ListenAndServe(fmt.Sprintf(":%d", s.port), nil); err != nil {
		return err
	}
	return nil
}

func writeErr(w http.ResponseWriter, errString string) {
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(errString))
}

func (s *MockTopologyServer) getTopology(w http.ResponseWriter, req *http.Request) {
	s.lock.Lock()
	defer s.lock.Unlock()

	asBytes, err := json.Marshal(s.topology)
	if err != nil {
		writeErr(w, "failed to convert topology to JSON")
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(asBytes)
}

func (s *MockTopologyServer) getNodesNum(w http.ResponseWriter, req *http.Request) {
	s.lock.Lock()
	defer s.lock.Unlock()

	num := len(s.topology.ListenAddr)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("%d", num)))
}

func (s *MockTopologyServer) getValidatorsForID(w http.ResponseWriter, req *http.Request) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if !req.URL.Query().Has("id") {
		writeErr(w, "this endpoint requires an 'id' query string param")
		return
	}
	qid := req.URL.Query().Get("id")
	var (
		validators []*validators.Validator
		ok         bool
	)

	id, err := strconv.Atoi(qid)
	if err != nil {
		writeErr(w, "id invalid")
		return
	}

	if validators, ok = s.topology.Connections[idx.ValidatorID(id)]; !ok {
		writeErr(w, "id not found")
		return
	}

	asBytes, err := json.Marshal(validators)
	if err != nil {
		writeErr(w, "failed to convert validators to JSON")
	}
	w.WriteHeader(http.StatusOK)
	w.Write(asBytes)

}

func (s *MockTopologyServer) setListenAddrForValidator(w http.ResponseWriter, req *http.Request) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if !req.URL.Query().Has("id") {
		writeErr(w, "this endpoint requires an 'id' query string param")
		return
	}
	if !req.URL.Query().Has("listen-addr") {
		writeErr(w, "this endpoint requires a 'listen-addr' query string param")
		return
	}
	qid := req.URL.Query().Get("id")
	qaddr := req.URL.Query().Get("listen-addr")

	id, err := strconv.Atoi(qid)
	if err != nil {
		writeErr(w, "id invalid")
		return
	}
	valID := idx.ValidatorID(id)
	s.topology.ListenAddr[valID] = qaddr
	for v := range s.topology.ListenAddr {
		if v == valID {
			continue
		}
		oneway := &validators.Validator{
			ID:         valID,
			ListenAddr: qaddr,
		}
		reverse := &validators.Validator{
			ID:         v,
			ListenAddr: s.topology.ListenAddr[v],
		}
		s.topology.Connections[v] = append(s.topology.Connections[v], oneway)
		s.topology.Connections[valID] = append(s.topology.Connections[valID], reverse)

	}
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("success"))
}

func (s *MockTopologyServer) getListenAddrForValidator(w http.ResponseWriter, req *http.Request) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if !req.URL.Query().Has("id") {
		writeErr(w, "this endpoint requires an 'id' query string param")
		return
	}
	qid := req.URL.Query().Get("id")
	id, err := strconv.Atoi(qid)
	if err != nil {
		writeErr(w, "id invalid")
		return
	}
	addr := s.topology.ListenAddr[idx.ValidatorID(id)]
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(addr))
}

func (s *MockTopologyServer) ready(w http.ResponseWriter, req *http.Request) {
	if s.isReady {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ready"))
		return
	}
	w.WriteHeader(http.StatusLocked)
	w.Write([]byte("not ready"))
}

func (s *MockTopologyServer) setReady(w http.ResponseWriter, req *http.Request) {
	s.isReady = true
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("success"))
}

func status(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("server running"))
}
