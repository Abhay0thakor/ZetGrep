package api

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
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
	json.NewEncoder(w).Encode(stats)
}

func (s *Server) handlePatterns(w http.ResponseWriter, r *http.Request) {
	p, _ := scanner.GetPatterns()
	json.NewEncoder(w).Encode(p)
}

func (s *Server) handleSavePattern(w http.ResponseWriter, r *http.Request) {
	var p models.Pattern
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}
	path := filepath.Join(s.svc.Config.PatternsDir, name+".json")
	b, _ := json.MarshalIndent(p, "", "  ")
	if err := os.WriteFile(path, b, 0644); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleDeletePattern(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	path := filepath.Join(s.svc.Config.PatternsDir, name+".json")
	os.Remove(path)
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleTools(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(s.svc.Tools)
}

func (s *Server) handleSaveTool(w http.ResponseWriter, r *http.Request) {
	var t models.Tool
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if t.ID == "" {
		http.Error(w, "ID required", http.StatusBadRequest)
		return
	}
	path := filepath.Join(s.svc.Config.ToolsDir, t.ID+".yaml")
	b, _ := yaml.Marshal(t)
	if err := os.WriteFile(path, b, 0644); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Reload tools in service
	s.svc.Tools = scanner.LoadToolsFrom(s.svc.Config.ToolsDir)
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleDeleteTool(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	path := filepath.Join(s.svc.Config.ToolsDir, id+".yaml")
	os.Remove(path)
	s.svc.Tools = scanner.LoadToolsFrom(s.svc.Config.ToolsDir)
	w.WriteHeader(http.StatusOK)
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
