package diagnostic

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"strconv"
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

	ticker  *time.Ticker
	closing chan struct{}
}

func NewSessionService() *SessionService {
	return &SessionService{
		sessions: &sessionsStore{
			sessions: make(map[uuid.UUID]*Session),
		},
	}
}

// TODO: implement
func (s *SessionService) Close() error {
	s.closing <- struct{}{}
	s.ticker.Stop()
	close(s.closing)
	return nil
}

func (s *SessionService) Open() error {
	ch := make(chan struct{}, 0)
	s.ticker = time.NewTicker(time.Second)

	s.routes = []httpd.Route{
		{
			Method:      "POST",
			Pattern:     sessionsPath,
			HandlerFunc: s.handleCreateSession,
		},
		{
			Method:      "GET",
			Pattern:     sessionsPath,
			HandlerFunc: s.handleSession,
		},
	}
	s.closing = ch

	if s.HTTPDService == nil {
		return errors.New("must set HTTPDService")
	}

	if err := s.HTTPDService.AddRoutes(s.routes); err != nil {
		return fmt.Errorf("failed to add routes: %v", err)
	}

	go func() {
		for {
			select {
			case <-s.ticker.C:
				if err := s.sessions.Prune(); err != nil {
					// TODO: log error
				}
			case <-ch:
				return
			}
		}
	}()
	return nil
}

func (s *SessionService) NewLogger() *sessionsLogger {
	return &sessionsLogger{
		store: s.sessions,
	}
}

func (s *SessionService) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	tags := []tag{}

	for k, v := range params {
		if len(v) != 1 {
			httpd.HttpError(w, "query params cannot contain duplicate pairs", true, http.StatusBadRequest)
			return
		}

		tags = append(tags, tag{key: k, value: v[0]})
	}

	session := s.sessions.Create(tags)
	u := fmt.Sprintf("%s%s?id=%s&page=%v", httpd.BasePath, sessionsPath, session.ID(), session.Page())

	header := w.Header()
	header.Add("Link", fmt.Sprintf("<%s>; rel=\"next\";", u))
	header.Add("Deadline", session.Deadline().UTC().String())
}

func (s *SessionService) handleSession(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()

	id := params.Get("id")
	if id == "" {
		httpd.HttpError(w, "missing id query param", true, http.StatusBadRequest)
		return
	}
	pageStr := params.Get("page")
	if pageStr == "" {
		httpd.HttpError(w, "missing page param", true, http.StatusBadRequest)
		return
	}
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		// TODO(desa): add some context to this error
		httpd.HttpError(w, err.Error(), true, http.StatusBadRequest)
		return
	}

	session, err := s.sessions.Get(id)
	if err != nil {
		// TODO(desa): add some context to this error
		httpd.HttpError(w, err.Error(), true, http.StatusBadRequest)
		return
	}

	p, err := session.GetPage(page)
	if err != nil {
		// TODO(desa): add some context to this error
		httpd.HttpError(w, err.Error(), true, http.StatusBadRequest)
		return
	}

	// TODO: add byte buffer pool here
	buf := bytes.NewBuffer(nil)
	// TODO: add support for JSON and logfmt encoding
	for _, l := range p {
		writeLogfmt(buf, l.Time, l.Level, l.Message, l.Context, l.Fields)
		//line.WriteTo(buf)
	}

	u := fmt.Sprintf("%s%s?id=%s&page=%v", httpd.BasePath, sessionsPath, session.ID(), session.Page())

	header := w.Header()
	header.Add("Link", fmt.Sprintf("<%s>; rel=\"next\";", u))
	header.Add("Deadline", session.Deadline().UTC().String())

	w.WriteHeader(http.StatusOK)
	w.Write(buf.Bytes())
	//w.Write([]byte("yah"))

	return
}
