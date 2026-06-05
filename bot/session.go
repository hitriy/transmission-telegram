package bot

import (
	"sync"

	"github.com/hitriy/transmission-telegram/rutracker"
)

const pageSize = 10

type SearchSession struct {
	Query       string
	Results     []rutracker.Torrent
	Page        int
	SelectedID  string
	ChatID      int64
	MessageID   int
}

type SessionStore struct {
	mu       sync.RWMutex
	sessions map[int]SearchSession // keyed by message ID
}

func NewSessionStore() *SessionStore {
	return &SessionStore{sessions: make(map[int]SearchSession)}
}

func (s *SessionStore) Set(msgID int, session SearchSession) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session.MessageID = msgID
	s.sessions[msgID] = session
}

func (s *SessionStore) Get(msgID int) (SearchSession, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.sessions[msgID]
	return session, ok
}

func (s *SessionStore) Delete(msgID int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, msgID)
}

func totalPages(count int) int {
	if count == 0 {
		return 1
	}
	pages := count / pageSize
	if count%pageSize != 0 {
		pages++
	}
	return pages
}

func pageItems(results []rutracker.Torrent, page int) []rutracker.Torrent {
	start := page * pageSize
	if start >= len(results) {
		return nil
	}
	end := start + pageSize
	if end > len(results) {
		end = len(results)
	}
	return results[start:end]
}
