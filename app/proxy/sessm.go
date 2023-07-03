package proxy

import "sync"

var sessManager = newSessionManager()

func AddCommonSwitch(s *SwitchSession) {
	sessManager.Add(s.ID, s)
}

func newSessionManager() *sessionManager {
	return &sessionManager{
		data: make(map[string]*SwitchSession),
	}
}

type sessionManager struct {
	data map[string]*SwitchSession
	sync.Mutex
}

func (s *sessionManager) Add(id string, sess *SwitchSession) {
	s.Lock()
	defer s.Unlock()
	s.data[id] = sess
}

func RemoveCommonSwitch(s *SwitchSession) {
	sessManager.Delete(s.ID)
}

func (s *sessionManager) Delete(id string) {
	s.Lock()
	defer s.Unlock()
	delete(s.data, id)
}
