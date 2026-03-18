package api

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Abhay0thakor/ZetGrep/pkg/models"
	"github.com/Abhay0thakor/ZetGrep/pkg/scanner"
)

//go:embed web/static
var staticFS embed.FS

type Server struct {
	addr       string
	staticFS   fs.FS
	svc        *scanner.ScannerService
}

func NewServer(addr string, svc *scanner.ScannerService) *Server {
	if svc == nil {
		svc, _ = scanner.NewScannerService(models.Config{})
	}
	return &Server{
		addr:       addr,
		staticFS:   GetStaticFS(),
		svc:        svc,
	}
}

func GetStaticFS() fs.FS {
	sub, _ := fs.Sub(staticFS, "web/static")
	return sub
}

func (s *Server) Start() error {
	if !strings.Contains(s.addr, ":") {
		s.addr = ":" + s.addr
	}
	cwd, _ := os.Getwd()
	os.MkdirAll(filepath.Join(cwd, "results"), 0755)
	
	mux := http.NewServeMux()
	mux.HandleFunc("/api/patterns", s.handlePatterns)
	mux.HandleFunc("/api/tools", s.handleTools)
	mux.HandleFunc("/api/tools/run", s.handleRunTool)
	mux.HandleFunc("/api/history", s.handleHistory)
	mux.HandleFunc("/api/history/load", s.handleLoadHistory)
	mux.HandleFunc("/api/scan", s.handleScan)
	
	mux.Handle("/", http.FileServer(http.FS(s.staticFS)))
	
	fmt.Printf("ZetGrep Intelligence Dashboard: http://localhost%s\n", s.addr)
	return http.ListenAndServe(s.addr, mux)
}

func (s *Server) handlePatterns(w http.ResponseWriter, r *http.Request) {
	p, _ := scanner.GetPatterns()
	json.NewEncoder(w).Encode(p)
}

func (s *Server) handleTools(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(s.svc.Tools)
}

func (s *Server) handleRunTool(w http.ResponseWriter, r *http.Request) {
	tid := r.URL.Query().Get("id")
	var res models.Result
	if err := json.NewDecoder(r.Body).Decode(&res); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	for _, t := range s.svc.Tools {
		if t.ID == tid {
			val, _ := t.Execute(res)
			json.NewEncoder(w).Encode(models.ToolOutput{ToolID: t.ID, Label: t.Field, Value: val})
			return
		}
	}
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	cwd, _ := os.Getwd()
	files, _ := filepath.Glob(filepath.Join(cwd, "results", "*.json"))
	var h []string
	for _, f := range files {
		h = append(h, filepath.Base(f))
	}
	json.NewEncoder(w).Encode(h)
}

func (s *Server) handleLoadHistory(w http.ResponseWriter, r *http.Request) {
	cwd, _ := os.Getwd()
	f := r.URL.Query().Get("file")
	b, err := os.ReadFile(filepath.Join(cwd, "results", f))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Write(b)
}

func (s *Server) handleScan(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	pNames := strings.Split(r.URL.Query().Get("patterns"), ",")
	saveAs := r.URL.Query().Get("saveAs")
	activeToolIDs := strings.Split(r.URL.Query().Get("tools"), ",")
	smart := r.URL.Query().Get("smart") == "true"
	entropy := r.URL.Query().Get("entropy") == "true"

	// JSONL Mode parameters
	jsonl := r.URL.Query().Get("jsonl") == "true"
	jsonlTarget := r.URL.Query().Get("jsonlTarget")
	jsonlId := r.URL.Query().Get("jsonlId")
	jsonlDecode := r.URL.Query().Get("jsonlDecode") == "true"

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	fl, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Create a temporary scanner service with request-specific JSONL config
	cfg := s.svc.Config
	if jsonl {
		cfg.Input.Format = "jsonl"
		cfg.Input.Target = jsonlTarget
		cfg.Input.ID = jsonlId
		cfg.Input.Decode = jsonlDecode
	} else {
		cfg.Input.Format = "" // Reset to default
	}

	reqSvc, _ := scanner.NewScannerService(cfg)

	opts := scanner.ScannerOptions{
		TargetPaths: []string{path},
		Patterns:    pNames,
		ToolIDs:     activeToolIDs,
		SmartMode:   smart,
		EntropyMode: entropy,
	}

	resultChan, err := reqSvc.RunScan(r.Context(), opts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var sf *os.File
	cwd, _ := os.Getwd()
	if saveAs != "" {
		if !strings.HasSuffix(saveAs, ".json") {
			saveAs += ".json"
		}
		sf, _ = os.Create(filepath.Join(cwd, "results", saveAs))
		if sf != nil {
			sf.WriteString("[")
		}
	}

	first := true
	counter := 0
	for res := range resultChan {
		counter++
		res.ID = counter
		data, _ := json.Marshal(res)
		fmt.Fprintf(w, "data: %s\n\n", data)
		fl.Flush()
		
		if sf != nil {
			if !first {
				sf.WriteString(",")
			}
			sf.Write(data)
			first = false
		}
		scanner.PutResult(res)
	}

	if sf != nil {
		sf.WriteString("]")
		sf.Close()
	}
	fmt.Fprintf(w, "event: done\ndata: {}\n\n")
	fl.Flush()
}
