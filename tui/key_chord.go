package tui

import "time"

type keyChordState struct {
	pending string
	at      time.Time
}

func (s *keyChordState) Pending() string {
	return s.pending
}

func (s *keyChordState) Set(keys string, now time.Time) {
	s.pending = keys
	s.at = now
}

func (s *keyChordState) Clear() {
	s.pending = ""
	s.at = time.Time{}
}

func (s *keyChordState) ClearExpired(now time.Time) {
	if s.pending != "" && now.Sub(s.at) > pendingKeyTimeout {
		s.Clear()
	}
}
