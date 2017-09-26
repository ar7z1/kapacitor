package diagnostic

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/influxdata/kapacitor/services/httpd"
	"github.com/influxdata/kapacitor/uuid"
)

const (
	sessionsPath = "/sessions"
)

//type Diagnostic interface {
//}

type SessionService struct {
	//diag   Diagnostic
	routes []httpd.Route

	sessions     SessionsStore
	HTTPDService interface {
		AddRoutes([]httpd.Route) error
	}
}

func NewSessionService() *SessionService {
	return &SessionService{
		sessions: &sessionsStore{
			sessions: make(map[uuid.UUID]*Session),
		},
	}
}

func (s *SessionService) Close() error {
	return nil
}

func (s *SessionService) Open() error {

	s.routes = []httpd.Route{
		{
			Method:      "GET",
			Pattern:     sessionsPath,
			HandlerFunc: s.handleSessions,
		},
	}

	if s.HTTPDService == nil {
		return errors.New("must set HTTPDService")
	}

	if err := s.HTTPDService.AddRoutes(s.routes); err != nil {
		return fmt.Errorf("failed to add routes: %v", err)
	}
	return nil
}

func (s *SessionService) NewLogger() *sessionsLogger {
	return &sessionsLogger{
		store: s.sessions,
	}
}

func (s *SessionService) handleSessions(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	tags := []tag{}

	for k, v := range params {
		if len(v) != 1 {
			httpd.HttpError(w, "query params cannot contain duplicate pairs", true, http.StatusBadRequest)
			return
		}

		tags = append(tags, tag{key: k, value: v[0]})
	}

	contentType := r.Header.Get("Content-Type")

	// TODO: do better verification of content type here
	session := s.sessions.Create(&httpWriteFlusher{w: w}, contentType, tags)
	defer s.sessions.Delete(session)

	header := w.Header()
	header.Add("Transfer-Encoding", "chunked")
	w.WriteHeader(http.StatusOK)

	ctx, cancel := context.WithTimeout(r.Context(), time.Minute)
	defer cancel()

	<-ctx.Done()
}

type WriteFlusher interface {
	Write([]byte) (int, error)
	Flush() error
}

type httpWriteFlusher struct {
	w http.ResponseWriter
}

func (h *httpWriteFlusher) Write(buf []byte) (int, error) {
	return h.w.Write(buf)
}
func (h *httpWriteFlusher) Flush() error {
	flusher, ok := h.w.(http.Flusher)
	if !ok {
		return errors.New("failed to coerce to http.Flusher")
	}

	flusher.Flush()

	return nil
}
