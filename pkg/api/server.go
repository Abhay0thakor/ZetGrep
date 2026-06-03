package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/Abhay0thakor/ZetGrep/pkg/models"
)

type Server struct {
	mu          sync.Mutex
	clients     map[chan *models.Result]bool
	port        int
}

func NewServer(port int) *Server {
	return &Server{
		clients: make(map[chan *models.Result]bool),
		port:    port,
	}
}

func (s *Server) Broadcast(res *models.Result) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for client := range s.clients {
		select {
		case client <- res:
		default:
			// Client channel full, skip to avoid blocking the engine
		}
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/events" {
		s.handleSSE(w, r)
		return
	}
	if r.URL.Path == "/" {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(indexHTML))
		return
	}
	http.NotFound(w, r)
}

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	clientChan := make(chan *models.Result, 100)
	s.mu.Lock()
	s.clients[clientChan] = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.clients, clientChan)
		s.mu.Unlock()
		close(clientChan)
	}()

	for {
		select {
		case <-r.Context().Done():
			return
		case res := <-clientChan:
			data, _ := json.Marshal(res)
			fmt.Fprintf(w, "data: %s\n\n", string(data))
			flusher.Flush()
		}
	}
}

func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.port)
	return http.ListenAndServe(addr, s)
}

const indexHTML = `
<!DOCTYPE html>
<html>
<head>
    <title>ZetGrep Live Dashboard</title>
    <style>
        body { font-family: monospace; background: #121212; color: #eee; margin: 20px; }
        .match { border-left: 3px solid #f1c40f; padding: 10px; margin-bottom: 10px; background: #1e1e1e; }
        .pattern { color: #f1c40f; font-weight: bold; }
        .file { color: #3498db; }
        .content { margin-top: 5px; color: #fff; }
        #stats { position: sticky; top: 0; background: #222; padding: 10px; border-bottom: 2px solid #444; margin-bottom: 20px; }
    </style>
</head>
<body>
    <div id="stats">
        Hits: <span id="hitCount">0</span> | Status: <span id="status" style="color: #2ecc71">Live Streaming</span>
    </div>
    <div id="results"></div>

    <script>
        let hitCount = 0;
        const results = document.getElementById('results');
        const countSpan = document.getElementById('hitCount');
        const evts = new EventSource('/events');

        evts.onmessage = (e) => {
            const res = JSON.parse(e.data);
            hitCount++;
            countSpan.innerText = hitCount;

            const div = document.createElement('div');
            div.className = 'match';
            div.innerHTML = "<div>[<span class='pattern'>" + res.pattern + "</span>] <span class='file'>" + res.file + ":" + res.line + "</span></div>" +
                            "<div class='content'>" + res.content + "</div>";
            results.prepend(div);
            if (results.children.length > 50) results.removeChild(results.lastChild);
        };

        evts.onerror = () => {
            document.getElementById('status').innerText = 'Disconnected';
            document.getElementById('status').style.color = '#e74c3c';
        };
    </script>
</body>
</html>
`
