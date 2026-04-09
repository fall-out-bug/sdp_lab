package tower

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

//go:embed all:web
var webFS embed.FS

// Serve starts the Control Tower HTTP server.
func Serve(projectRoot, port string) error {
	if port == "" {
		port = "8090"
	}

	mux := http.NewServeMux()
	t := &handler{projectRoot: projectRoot}

	mux.HandleFunc("GET /{$}", t.handleBoard)
	mux.HandleFunc("GET /card/{id}", t.handleCard)
	mux.HandleFunc("GET /card/{id}/detail", t.handleCardPartial) // HTMX partial
	mux.HandleFunc("POST /card/{id}/clarify", t.handleAction("clarify"))
	mux.HandleFunc("POST /card/{id}/approve", t.handleAction("approve"))
	mux.HandleFunc("POST /card/{id}/close", t.handleAction("close"))
	mux.HandleFunc("POST /card/{id}/reopen", t.handleAction("reopen"))

	addr := ":" + port
	slog.Info("Control Tower", "url", "http://localhost"+addr)
	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	return srv.ListenAndServe()
}

type handler struct {
	projectRoot string
}

func (h *handler) handleBoard(w http.ResponseWriter, r *http.Request) {
	data, err := h.bdList(true)
	if err != nil {
		http.Error(w, "bd list: "+err.Error(), 500)
		return
	}

	filter := BoardFilter{
		Project: r.URL.Query().Get("project"),
		Phase:   r.URL.Query().Get("phase"),
		Query:   r.URL.Query().Get("q"),
	}

	board, err := LoadBoard(data, filter)
	if err != nil {
		http.Error(w, "parse: "+err.Error(), 500)
		return
	}

	// Load evidence previews for visible cards
	for i := range board.Columns {
		for j := range board.Columns[i].Cards {
			cv := &board.Columns[i].Cards[j]
			cv.Evidence = LoadEvidencePreview(cv.ID, h.projectRoot)
		}
	}

	// Pass filter state for form
	type boardView struct {
		*BoardData
		QueryParam    string
		FilterProject string
		FilterPhase   string
	}
	render(w, "board", boardView{
		BoardData:     board,
		QueryParam:    filter.Query,
		FilterProject: filter.Project,
		FilterPhase:   filter.Phase,
	})
}

func (h *handler) handleCard(w http.ResponseWriter, r *http.Request) {
	detail, err := h.loadDetail(r.PathValue("id"))
	if err != nil {
		http.Error(w, err.Error(), 404)
		return
	}
	render(w, "card", detail)
}

func (h *handler) handleCardPartial(w http.ResponseWriter, r *http.Request) {
	// HTMX partial — returns just the card detail fragment
	detail, err := h.loadDetail(r.PathValue("id"))
	if err != nil {
		http.Error(w, err.Error(), 404)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl := template.Must(template.New("partial").Funcs(funcMap()).ParseFS(webFS, "web/card_detail.html"))
	if err := tmpl.Execute(w, detail); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *handler) handleAction(action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cardID := r.PathValue("id")
		var cmd *exec.Cmd
		bin := os.Getenv("OPENCODE_BIN")

		switch action {
		case "clarify":
			cmd = exec.CommandContext(context.Background(), "sdp", "clarify", cardID)
			cmd.Env = append(os.Environ(), "OPENCODE_BIN="+bin)
		case "approve":
			cmd = exec.CommandContext(context.Background(), "sdp", "approve-plan", cardID)
		case "close":
			cmd = exec.CommandContext(context.Background(), "bd", "close", cardID)
		case "reopen":
			cmd = exec.CommandContext(context.Background(), "bd", "reopen", cardID)
		}
		if cmd == nil {
			http.Error(w, "unknown action", 400)
			return
		}
		cmd.Dir = h.projectRoot
		out, err := cmd.CombinedOutput()
		if err != nil {
			slog.Error("action failed", "action", action, "card", cardID, "error", err, "output", string(out))
		} else {
			slog.Info("action completed", "action", action, "card", cardID)
		}

		// HTMX: if requested as partial, return the updated card
		if r.Header.Get("HX-Request") == "true" {
			w.Header().Set("HX-Redirect", "/card/"+cardID)
			return
		}
		http.Redirect(w, r, "/card/"+cardID, http.StatusSeeOther)
	}
}

func (h *handler) loadDetail(cardID string) (*CardDetail, error) {
	data, err := h.bdShow(cardID)
	if err != nil {
		return nil, fmt.Errorf("bd show: %w", err)
	}
	detail, err := LoadCardDetail(data, cardID, h.projectRoot)
	if err != nil {
		return nil, err
	}
	return detail, nil
}

func (h *handler) bdList(all bool) ([]byte, error) {
	args := []string{"list", "--json", "-n", "0"}
	if all {
		args = append(args, "--all")
	}
	cmd := exec.CommandContext(context.Background(), "bd", args...)
	cmd.Dir = h.projectRoot
	return cmd.Output()
}

func (h *handler) bdShow(id string) ([]byte, error) {
	cmd := exec.CommandContext(context.Background(), "bd", "show", id, "--json")
	cmd.Dir = h.projectRoot
	return cmd.Output()
}

func render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl := template.Must(template.New("layout").Funcs(funcMap()).ParseFS(webFS, "web/layout.html", "web/"+name+".html", "web/card_detail.html"))
	if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		slog.Error("render error", "template", name, "error", err)
	}
}

func funcMap() template.FuncMap {
	return template.FuncMap{
		"scorePct": func(s float64) string { return fmt.Sprintf("%.0f%%", s*100) },
		"shortCommit": func(s string) string {
			if len(s) > 7 {
				return s[:7]
			}
			return s
		},
		"shortTime": func(s string) string {
			t, err := time.Parse(time.RFC3339, s)
			if err != nil {
				return ""
			}
			return t.Format("15:04 Jan 02")
		},
		"hasPrefix": strings.HasPrefix,
		"contains":  strings.Contains,
		"priorityLabel": func(p int) string {
			if p <= 1 {
				return "P0-P1"
			}
			return fmt.Sprintf("P%d", p)
		},
		"priorityColor": func(p int) string {
			if p <= 1 {
				return "bg-red-500/20 text-red-400 border-red-500/30"
			}
			if p == 2 {
				return "bg-orange-500/20 text-orange-400 border-orange-500/30"
			}
			return "bg-gray-700/50 text-gray-400 border-gray-600/30"
		},
		"phaseColor": func(phase string) string {
			m := map[string]string{
				"clarify":  "bg-blue-500/20 text-blue-400",
				"plan":     "bg-indigo-500/20 text-indigo-400",
				"build":    "bg-amber-500/20 text-amber-400",
				"evaluate": "bg-purple-500/20 text-purple-400",
			}
			if c, ok := m[phase]; ok {
				return c
			}
			return "bg-gray-700/50 text-gray-500"
		},
		"verdictColor": func(v string) string {
			switch v {
			case "pass":
				return "text-green-400"
			case "fail":
				return "text-red-400"
			case "blocked":
				return "text-red-300"
			case "needs_review":
				return "text-yellow-400"
			default:
				return "text-gray-500"
			}
		},
		"verdictIcon": func(v string) string {
			switch v {
			case "pass":
				return "✅"
			case "fail":
				return "❌"
			case "blocked":
				return "🚫"
			case "needs_review":
				return "⚠️"
			default:
				return "—"
			}
		},
		"strSlice": func(s []string) template.HTML {
			if len(s) == 0 {
				return ""
			}
			var b strings.Builder
			for _, v := range s {
				fmt.Fprintf(&b, `<span class="inline-block px-1.5 py-0.5 bg-gray-800 rounded text-xs mr-1 mb-1">%s</span>`, v)
			}
			return template.HTML(b.String())
		},
		"criteriaGrid": func(m map[string]bool) template.HTML {
			if m == nil {
				return `<span class="text-xs text-gray-600">—</span>`
			}
			var b strings.Builder
			for k, v := range m {
				icon := "✅"
				color := "text-green-500"
				if !v {
					icon = "❌"
					color = "text-red-400"
				}
				fmt.Fprintf(&b, `<span class="inline-flex items-center gap-1 text-xs %s mr-3">%s %s</span>`, color, icon, k)
			}
			return template.HTML(b.String())
		},
		"labelStyle": func(label string) string {
			if strings.HasPrefix(label, "sdp:project-") {
				return "bg-cyan-500/15 text-cyan-400 border-cyan-500/25"
			}
			if strings.HasPrefix(label, "sdp:gate:") {
				return "bg-rose-500/15 text-rose-400 border-rose-500/25"
			}
			if strings.HasPrefix(label, "sdp:phase-") {
				return "bg-violet-500/15 text-violet-400 border-violet-500/25"
			}
			return "bg-gray-700/40 text-gray-400 border-gray-600/30"
		},
		"jsonIndent": func(v any) string {
			b, _ := json.MarshalIndent(v, "", "  ")
			return string(b)
		},
		"add": func(a, b int) int { return a + b },
	}
}
