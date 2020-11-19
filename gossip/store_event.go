package gossip

/*
	In LRU cache data stored like pointer
*/

import (
	"bytes"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/Fantom-foundation/go-opera/inter"
)

// DelEvent deletes event.
func (s *Store) DelEvent(id hash.Event) {
	key := id.Bytes()

	err := s.table.Events.Delete(key)
	if err != nil {
		s.Log.Crit("Failed to delete key", "err", err)
	}

	// Remove from LRU cache.
	if s.cache.Events != nil {
		s.cache.Events.Remove(id)
		s.cache.EventsHeaders.Remove(id)
	}
}

// SetEvent stores event.
func (s *Store) SetEvent(e *inter.EventPayload) {
	key := e.ID().Bytes()

	s.set(s.table.Events, key, e)

	// Add to LRU cache.
	if s.cache.Events != nil {
		s.cache.Events.Add(e.ID(), e)
		eh := e.Event
		s.cache.EventsHeaders.Add(e.ID(), &eh)
	}
}

// GetEventPayload returns stored event.
func (s *Store) GetEventPayload(id hash.Event) *inter.EventPayload {
	// Get event from LRU cache first.
	if ev, ok := s.cache.Events.Get(id); ok {
		return ev.(*inter.EventPayload)
	}

	key := id.Bytes()
	w, _ := s.get(s.table.Events, key, &inter.EventPayload{}).(*inter.EventPayload)

	// Put event to LRU cache.
	if w != nil {
		s.cache.Events.Add(id, w)
		eh := w.Event
		s.cache.EventsHeaders.Add(id, &eh)
	}

	return w
}

// GetEventPayload returns stored event.
func (s *Store) GetEvent(id hash.Event) *inter.Event {

	// Get event from LRU cache first.
	if ev, ok := s.cache.EventsHeaders.Get(id); ok {
		return ev.(*inter.Event)
	}

	key := id.Bytes()
	w, _ := s.get(s.table.Events, key, &inter.EventPayload{}).(*inter.EventPayload)
	if w == nil {
		return nil
	}
	eh := w.Event

	// Put event to LRU cache.
	s.cache.Events.Add(id, w)
	s.cache.EventsHeaders.Add(id, &eh)

	return &eh
}

func (s *Store) forEachEvent(it ethdb.Iterator, onEvent func(event *inter.EventPayload) bool) {
	for it.Next() {
		event := &inter.EventPayload{}
		err := rlp.DecodeBytes(it.Value(), event)
		if err != nil {
			s.Log.Crit("Failed to decode event", "err", err)
		}

		if !onEvent(event) {
			return
		}
	}
}

func (s *Store) ForEachEpochEvent(epoch idx.Epoch, onEvent func(event *inter.EventPayload) bool) {
	it := s.table.Events.NewIterator(epoch.Bytes(), nil)
	defer it.Release()
	s.forEachEvent(it, onEvent)
}

func (s *Store) ForEachEvent(start idx.Epoch, onEvent func(event *inter.EventPayload) bool) {
	it := s.table.Events.NewIterator(nil, start.Bytes())
	defer it.Release()
	s.forEachEvent(it, onEvent)
}

func (s *Store) ForEachEventRLP(start idx.Epoch, onEvent func(key hash.Event, event rlp.RawValue) bool) {
	it := s.table.Events.NewIterator(nil, start.Bytes())
	defer it.Release()
	for it.Next() {
		if !onEvent(hash.BytesToEvent(it.Key()), it.Value()) {
			return
		}
	}
}

func (s *Store) FindEventHashes(epoch idx.Epoch, lamport idx.Lamport, hashPrefix []byte) hash.Events {
	prefix := bytes.NewBuffer(epoch.Bytes())
	prefix.Write(lamport.Bytes())
	prefix.Write(hashPrefix)
	res := make(hash.Events, 0, 10)

	it := s.table.Events.NewIterator(prefix.Bytes(), nil)
	defer it.Release()
	for it.Next() {
		res = append(res, hash.BytesToEvent(it.Key()))
	}

	return res
}

// GetEventPayloadRLP returns stored event. Serialized.
func (s *Store) GetEventPayloadRLP(id hash.Event) rlp.RawValue {
	key := id.Bytes()

	data, err := s.table.Events.Get(key)
	if err != nil {
		s.Log.Crit("Failed to get key-value", "err", err)
	}
	return data
}

// HasEvent returns true if event exists.
func (s *Store) HasEvent(h hash.Event) bool {
	return s.has(s.table.Events, h.Bytes())
}