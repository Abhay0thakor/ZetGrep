package api

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Abhay0thakor/ZetGrep/pkg/models"
	"github.com/Abhay0thakor/ZetGrep/pkg/scanner"
	"gopkg.in/yaml.v3"
)

//go:embed web/static
var staticFS embed.FS

type Server struct {
	addr     string
	staticFS fs.FS
	svc      *scanner.ScannerService
}

func NewServer(addr string, svc *scanner.ScannerService) *Server {
	if svc == nil {
		svc, _ = scanner.NewScannerService(models.Config{})
	}
	return &Server{
		addr:     addr,
		staticFS: GetStaticFS(),
		svc:      svc,
	}
}

func GetStaticFS() fs.FS {
	sub, _ := fs.Sub(staticFS, "web/static")
	return sub
}

func (s *Server) jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			slog.Error("Failed to encode JSON response", "error", err)
		}
	}
}

func (s *Server) errorResponse(w http.ResponseWriter, status int, message string) {
	s.jsonResponse(w, status, map[string]string{"error": message})
}

func (s *Server) Start() error {
	if !strings.Contains(s.addr, ":") {
		s.addr = ":" + s.addr
	}
	cwd, _ := os.Getwd()
	os.MkdirAll(filepath.Join(cwd, "results"), 0755)

	mux := http.NewServeMux()
	// Existing endpoints
	mux.HandleFunc("/api/patterns", s.handlePatterns)
	mux.HandleFunc("/api/tools", s.handleTools)
	mux.HandleFunc("/api/tools/run", s.handleRunTool)
	mux.HandleFunc("/api/history", s.handleHistory)
	mux.HandleFunc("/api/history/load", s.handleLoadHistory)
	mux.HandleFunc("/api/scan", s.handleScan)
	mux.HandleFunc("/api/ls", s.handleLs)

	// New Mission Control 2.0 endpoints
	mux.HandleFunc("/api/stats", s.handleStats)
	mux.HandleFunc("/api/patterns/save", s.handleSavePattern)
	mux.HandleFunc("/api/patterns/delete", s.handleDeletePattern)
	mux.HandleFunc("/api/tools/save", s.handleSaveTool)
	mux.HandleFunc("/api/tools/delete", s.handleDeleteTool)

	mux.HandleFunc("/logo.svg", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "logo.svg")
	})

	mux.Handle("/docs/", http.StripPrefix("/docs/", http.FileServer(http.Dir("docs"))))
	mux.Handle("/", http.FileServer(http.FS(s.staticFS)))

	fmt.Printf("ZetGrep Mission Control: http://localhost%s\n", s.addr)
	return http.ListenAndServe(s.addr, mux)
}

// handleStats provides real-time system metrics
func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	stats := map[string]interface{}{
		"ram":        m.Alloc / 1024 / 1024, // MB
		"num_cpu":    runtime.NumCPU(),
		"goroutines": runtime.NumGoroutine(),
	}
	s.jsonResponse(w, http.StatusOK, stats)
}

func (s *Server) handlePatterns(w http.ResponseWriter, r *http.Request) {
	p, err := scanner.GetPatterns(s.svc.Config.PatternsDir)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Failed to get patterns")
		return
	}
	s.jsonResponse(w, http.StatusOK, p)
}

func (s *Server) handleSavePattern(w http.ResponseWriter, r *http.Request) {
	var p models.Pattern
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	name := r.URL.Query().Get("name")
	if name == "" {
		s.errorResponse(w, http.StatusBadRequest, "Parameter 'name' is required")
		return
	}
	if strings.Contains(name, "..") || strings.Contains(name, "/") {
		s.errorResponse(w, http.StatusBadRequest, "Invalid pattern name")
		return
	}
	path := filepath.Join(s.svc.Config.PatternsDir, name+".json")
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Failed to encode pattern")
		return
	}
	if err := os.WriteFile(path, b, 0644); err != nil {
		slog.Error("Failed to save pattern", "path", path, "error", err)
		s.errorResponse(w, http.StatusInternalServerError, "Failed to save pattern file")
		return
	}
	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleDeletePattern(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		s.errorResponse(w, http.StatusBadRequest, "Parameter 'name' is required")
		return
	}
	if strings.Contains(name, "..") || strings.Contains(name, "/") {
		s.errorResponse(w, http.StatusBadRequest, "Invalid pattern name")
		return
	}
	path := filepath.Join(s.svc.Config.PatternsDir, name+".json")
	if err := os.Remove(path); err != nil {
		slog.Error("Failed to delete pattern", "path", path, "error", err)
		s.errorResponse(w, http.StatusInternalServerError, "Failed to delete pattern file")
		return
	}
	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleTools(w http.ResponseWriter, r *http.Request) {
	s.jsonResponse(w, http.StatusOK, s.svc.Tools)
}

func (s *Server) handleSaveTool(w http.ResponseWriter, r *http.Request) {
	var t models.Tool
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if t.ID == "" {
		s.errorResponse(w, http.StatusBadRequest, "Tool ID is required")
		return
	}
	if strings.Contains(t.ID, "..") || strings.Contains(t.ID, "/") {
		s.errorResponse(w, http.StatusBadRequest, "Invalid tool ID")
		return
	}
	path := filepath.Join(s.svc.Config.ToolsDir, t.ID+".yaml")
	b, err := yaml.Marshal(t)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Failed to encode tool")
		return
	}
	if err := os.WriteFile(path, b, 0644); err != nil {
		slog.Error("Failed to save tool", "path", path, "error", err)
		s.errorResponse(w, http.StatusInternalServerError, "Failed to save tool file")
		return
	}
	// Reload tools in service
	s.svc.Tools, _ = scanner.LoadToolsFrom(s.svc.Config.ToolsDir)
	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleDeleteTool(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		s.errorResponse(w, http.StatusBadRequest, "Parameter 'id' is required")
		return
	}
	if strings.Contains(id, "..") || strings.Contains(id, "/") {
		s.errorResponse(w, http.StatusBadRequest, "Invalid tool ID")
		return
	}
	path := filepath.Join(s.svc.Config.ToolsDir, id+".yaml")
	if err := os.Remove(path); err != nil {
		slog.Error("Failed to delete tool", "path", path, "error", err)
		s.errorResponse(w, http.StatusInternalServerError, "Failed to delete tool file")
		return
	}
	s.svc.Tools, _ = scanner.LoadToolsFrom(s.svc.Config.ToolsDir)
	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleRunTool(w http.ResponseWriter, r *http.Request) {
	tid := r.URL.Query().Get("id")
	if tid == "" {
		s.errorResponse(w, http.StatusBadRequest, "Parameter 'id' is required")
		return
	}
	var res models.Result
	if err := json.NewDecoder(r.Body).Decode(&res); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	for _, t := range s.svc.Tools {
		if t.ID == tid {
			val, err := t.Execute(res)
			if err != nil {
				s.errorResponse(w, http.StatusInternalServerError, err.Error())
				return
			}
			s.jsonResponse(w, http.StatusOK, models.ToolOutput{ToolID: t.ID, Label: t.Field, Value: val})
			return
		}
	}
	s.errorResponse(w, http.StatusNotFound, "Tool not found")
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	cwd, _ := os.Getwd()
	files, err := filepath.Glob(filepath.Join(cwd, "results", "*.json"))
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Failed to list history")
		return
	}
	var h []string
	for _, f := range files {
		h = append(h, filepath.Base(f))
	}
	s.jsonResponse(w, http.StatusOK, h)
}

func (s *Server) handleLoadHistory(w http.ResponseWriter, r *http.Request) {
	cwd, _ := os.Getwd()
	f := r.URL.Query().Get("file")
	if f == "" {
		s.errorResponse(w, http.StatusBadRequest, "Parameter 'file' is required")
		return
	}
	if strings.Contains(f, "..") || strings.Contains(f, "/") {
		s.errorResponse(w, http.StatusBadRequest, "Invalid filename")
		return
	}
	path := filepath.Join(cwd, "results", f)
	b, err := os.ReadFile(path)
	if err != nil {
		s.errorResponse(w, http.StatusNotFound, "History file not found")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

func (s *Server) handleScan(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	pNames := strings.Split(r.URL.Query().Get("patterns"), ",")
	saveAs := r.URL.Query().Get("saveAs")
	activeToolIDs := strings.Split(r.URL.Query().Get("tools"), ",")
	smart := r.URL.Query().Get("smart") == "true"
	entropy := r.URL.Query().Get("entropy") == "true"

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

	cfg := s.svc.Config
	if jsonl {
		cfg.Input.Format = "jsonl"
		cfg.Input.Target = jsonlTarget
		cfg.Input.ID = jsonlId
		cfg.Input.Decode = jsonlDecode
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

func (s *Server) handleLs(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		path = "."
	}
	
	// Basic Path Traversal Protection
	if strings.Contains(path, "..") {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	dir := filepath.Dir(path)
	base := filepath.Base(path)
	if strings.HasSuffix(path, "/") || path == "." {
		dir = path
		base = ""
	}
	files, err := os.ReadDir(dir)
	if err != nil {
		json.NewEncoder(w).Encode([]string{})
		return
	}
	var suggestions []string
	for _, f := range files {
		name := f.Name()
		if base == "" || strings.HasPrefix(name, base) {
			full := filepath.Join(dir, name)
			if f.IsDir() {
				full += "/"
			}
			suggestions = append(suggestions, full)
		}
	}
	if suggestions == nil {
		suggestions = []string{}
	}
	json.NewEncoder(w).Encode(suggestions)
}
