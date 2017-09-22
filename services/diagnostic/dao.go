package diagnostic

import (
	"errors"
	"sync"
	"time"

	"github.com/influxdata/kapacitor/uuid"
)

const (
	pageSize = 10
	// TODO: what to make this value
	sessionExipryDuration = 10 * time.Second
)

type SessionsStore interface {
	Create(tags []tag) *Session
	Get(id string) (*Session, error)
	Delete(id string) error
	Prune() error
	Each(func(*Session))
}

type sessionsStore struct {
	mu       sync.RWMutex
	sessions map[uuid.UUID]*Session
}

func (kv *sessionsStore) Create(tags []tag) *Session {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	s := &Session{
		id:       uuid.New(),
		deadline: time.Now().Add(sessionExipryDuration),
		tags:     tags,
		queue:    &Queue{},
	}

	kv.sessions[s.id] = s

	// TODO: register with Diagnostic service
	return s
}

func (kv *sessionsStore) Delete(id string) error {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	s, err := kv.get(id)
	if err != nil {
		return err
	}

	if err := s.Close(); err != nil {
		return err
	}

	delete(kv.sessions, s.id)

	return nil
}

func (kv *sessionsStore) Each(fn func(*Session)) {
	kv.mu.RLock()
	defer kv.mu.RUnlock()
	for _, s := range kv.sessions {
		fn(s)
	}
}

func (kv *sessionsStore) Prune() error {
	ids := []uuid.UUID{}
	kv.mu.RLock()
	now := time.Now()
	for _, s := range kv.sessions {
		if now.After(s.deadline) {
			ids = append(ids, s.id)
		}
	}
	kv.mu.RUnlock()

	errs := []error{}
	for _, id := range ids {
		// TODO: maybe change function signature of delete
		if err := kv.Delete(id.String()); err != nil {
			// TODO log error
			errs = append(errs, err)
		}
	}

	return nil
}

func (kv *sessionsStore) Get(id string) (*Session, error) {
	kv.mu.RLock()
	defer kv.mu.RUnlock()
	s, err := kv.get(id)
	if err != nil {
		return nil, err
	}

	if time.Now().After(s.deadline) {
		return nil, errors.New("session expired")
	}

	return s, nil
}

func (kv *sessionsStore) get(id string) (*Session, error) {
	sid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	s, ok := kv.sessions[sid]
	if !ok {
		return nil, errors.New("session not found")
	}

	return s, nil
}

type sessionsLogger struct {
	store   SessionsStore
	context []Field
}

func (s *sessionsLogger) Error(msg string, ctx ...Field) {
	s.store.Each(func(sn *Session) {
		sn.Error(msg, s.context, ctx)
	})
}

func (s *sessionsLogger) Warn(msg string, ctx ...Field) {
	s.store.Each(func(sn *Session) {
		sn.Warn(msg, s.context, ctx)
	})
}

func (s *sessionsLogger) Debug(msg string, ctx ...Field) {
	s.store.Each(func(sn *Session) {
		sn.Warn(msg, s.context, ctx)
	})
}

func (s *sessionsLogger) Info(msg string, ctx ...Field) {
	s.store.Each(func(sn *Session) {
		sn.Info(msg, s.context, ctx)
	})
}

func (s *sessionsLogger) With(ctx ...Field) Logger {
	// TODO: this needs some kind of locking
	return &sessionsLogger{
		store:   s.store,
		context: append(s.context, ctx...),
	}
}

type tag struct {
	key   string
	value string
}

type Session struct {
	mu       sync.RWMutex
	id       uuid.UUID
	page     int
	deadline time.Time

	tags []tag

	queue *Queue
}

func (s *Session) ID() string {
	return s.id.String()
}

func (s *Session) Deadline() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.deadline
}

func (s *Session) Page() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.page
}

func (s *Session) GetPage(page int) ([]*Data, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if page != s.page {
		return nil, errors.New("bad page value")
	}
	s.page++
	s.deadline = s.deadline.Add(sessionExipryDuration)

	//l := make([]*Data, 0, pageSize)
	l := []*Data{}
	for i := 0; i < pageSize; i++ {
		if s.queue.Len() == 0 {
			break
		}
		if d := s.queue.Dequeue(); d != nil {
			l = append(l, d)
		}
	}

	return l, nil
}

// TODO: implement closing logic here
func (s *Session) Close() error {
	return nil
}

func (s *Session) Error(msg string, context, fields []Field) {
	if match(s.tags, msg, "error", context, fields) {
		s.queue.Enqueue(&Data{
			Time:    time.Now(),
			Message: msg,
			Level:   "info",
			Context: context,
			Fields:  fields,
		})
	}
}

func (s *Session) Warn(msg string, context, fields []Field) {
	if match(s.tags, msg, "warn", context, fields) {
		s.queue.Enqueue(&Data{
			Time:    time.Now(),
			Message: msg,
			Level:   "info",
			Context: context,
			Fields:  fields,
		})
	}
}

func (s *Session) Debug(msg string, context, fields []Field) {
	if match(s.tags, msg, "debug", context, fields) {
		s.queue.Enqueue(&Data{
			Time:    time.Now(),
			Message: msg,
			Level:   "info",
			Context: context,
			Fields:  fields,
		})
	}
}

func (s *Session) Info(msg string, context, fields []Field) {
	if match(s.tags, msg, "info", context, fields) {
		s.queue.Enqueue(&Data{
			Time:    time.Now(),
			Message: msg,
			Level:   "info",
			Context: context,
			Fields:  fields,
		})
	}
}

// TODO: check level and msg
func match(tags []tag, msg, level string, context, fields []Field) bool {
	ctr := 0
Loop:
	for _, t := range tags {
		if t.key == "msg" && t.value == msg {
			ctr++
			continue Loop
		}
		if t.key == "lvl" && t.value == level {
			ctr++
			continue Loop
		}
		for _, c := range context {
			if c.Match(t.key, t.value) {
				ctr++
				continue Loop
			}
		}
		for _, f := range fields {
			if f.Match(t.key, t.value) {
				ctr++
				continue Loop
			}
		}
	}

	return len(tags) == ctr
}
