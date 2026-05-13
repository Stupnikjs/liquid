package cache

// changer map par array avec HF triée

func (s *MarketStore) AllPosLen() int {
	s.mu.RLock()
	sum := 0
	for _, m := range s.markets {
		sum += len(m.Positions)
	}
	s.mu.RUnlock()
	return sum

}

func (s *MarketStore) Range(fn func(id [32]byte)) {
	ids := s.Ids()
	for _, id := range ids {
		fn(id)
	}
}

func (s *MarketStore) Ids() [][32]byte {
	s.mu.RLock()
	ids := make([][32]byte, 0, len(s.markets))
	for id, m := range s.markets {
		if !m.Canceled {
			ids = append(ids, id)
		}
	}
	s.mu.RUnlock()
	return ids

}

func (s *MarketStore) Update(id [32]byte, fn func(m *Market)) {
	s.mu.RLock()
	m := s.markets[id]
	s.mu.RUnlock()

	if m == nil {
		return
	}

	m.Mu.Lock()
	fn(m)
	m.Mu.Unlock()
}
