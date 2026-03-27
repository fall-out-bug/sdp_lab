package tower

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

//go:embed all:web
var webFS embed.FS

// Serve starts the Control Tower HTTP server.
func Serve(projectRoot, port string) error {
	if port == "" {
		port = "8090"
	}

	mux := http.NewServeMux()
	t := towerHandler{projectRoot: projectRoot}

	// Static assets
	mux.Handle("GET /static/", http.FileServer(http.FS(webFS)))

	// Pages
	mux.HandleFunc("GET /{$}", t.handleBoard)
	mux.HandleFunc("GET /card/{id}", t.handleCard)
	mux.HandleFunc("POST /card/{id}/clarify", t.handleAction("clarify"))
	mux.HandleFunc("POST /card/{id}/approve", t.handleAction("approve"))
	mux.HandleFunc("POST /card/{id}/close", t.handleAction("close"))
	mux.HandleFunc("POST /card/{id}/reopen", t.handleAction("reopen"))

	addr := ":" + port
	log.Printf("⚙️ Control Tower: http://localhost%s", addr)
	return http.ListenAndServe(addr, mux)
}

type towerHandler struct {
	projectRoot string
}

func (t *towerHandler) handleBoard(w http.ResponseWriter, r *http.Request) {
	data, err := t.bdList(true)
	if err != nil {
		http.Error(w, "bd list: "+err.Error(), 500)
		return
	}

	board, err := LoadBoard(data)
	if err != nil {
		http.Error(w, "parse: "+err.Error(), 500)
		return
	}

	render(w, "board", board)
}

func (t *towerHandler) handleCard(w http.ResponseWriter, r *http.Request) {
	cardID := r.PathValue("id")

	data, err := t.bdShow(cardID)
	if err != nil {
		http.Error(w, "bd show: "+err.Error(), 500)
		return
	}

	var issues []map[string]any
	if err := json.Unmarshal(data, &issues); err != nil {
		http.Error(w, "parse: "+err.Error(), 500)
		return
	}

	if len(issues) == 0 {
		http.Error(w, "card not found", 404)
		return
	}

	issue := issues[0]
	detail, _ := LoadCardDetail(cardID, t.projectRoot)

	// Merge issue data into detail
	detail.ID = cardID
	detail.Title = strVal(issue, "title")
	detail.Description = strVal(issue, "description")
	detail.Status = strVal(issue, "status")
	detail.Priority = intVal(issue, "priority")
	if labels, ok := issue["labels"].([]any); ok {
		for _, l := range labels {
			detail.Labels = append(detail.Labels, fmt.Sprint(l))
		}
	}
	for _, l := range detail.Labels {
		if strings.HasPrefix(l, "sdp:phase-") {
			detail.Phase = strings.TrimPrefix(l, "sdp:phase-")
		}
		if strings.HasPrefix(l, "sdp:project-") {
			detail.Project = strings.TrimPrefix(l, "sdp:project-")
		}
	}

	render(w, "card", detail)
}

func (t *towerHandler) handleAction(action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cardID := r.PathValue("id")
		var cmd *exec.Cmd

		switch action {
		case "clarify":
			bin := os.Getenv("OPENCODE_BIN")
			if bin == "" {
				bin = "opencode"
			}
			cmd = exec.Command("sdp", "clarify", cardID)
			cmd.Env = append(os.Environ(), "OPENCODE_BIN="+bin)
		case "approve":
			cmd = exec.Command("sdp", "approve-plan", cardID)
		case "close":
			cmd = exec.Command("bd", "close", cardID)
		case "reopen":
			cmd = exec.Command("bd", "reopen", cardID)
		}

		if cmd == nil {
			http.Error(w, "unknown action", 400)
			return
		}
		cmd.Dir = t.projectRoot
		out, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("action %s on %s: %v\n%s", action, cardID, err, out)
		}

		http.Redirect(w, r, "/card/"+cardID, http.StatusSeeOther)
	}
}

func (t *towerHandler) bdList(all bool) ([]byte, error) {
	args := []string{"list", "--json", "-n", "0"}
	if all {
		args = append(args, "--all")
	}
	cmd := exec.Command("bd", args...)
	cmd.Dir = t.projectRoot
	return cmd.Output()
}

func (t *towerHandler) bdShow(id string) ([]byte, error) {
	cmd := exec.Command("bd", "show", id, "--json")
	cmd.Dir = t.projectRoot
	return cmd.Output()
}

func render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl := template.Must(template.New("layout").Funcs(template.FuncMap{
		"phaseTag": phaseTag,
		"scorePct": func(s float64) string { return fmt.Sprintf("%.0f%%", s*100) },
		"priorityBadge": priorityBadge,
		"labelTag": labelTag,
	}).ParseFS(webFS, "web/layout.html", "web/"+name+".html"))
	if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		log.Printf("render error: %v", err)
	}
}

func phaseTag(phase string) template.HTML {
	if phase == "" {
		return `<span class="px-2 py-0.5 bg-gray-700 rounded text-xs">—</span>`
	}
	classes := map[string]string{
		"clarify": "bg-blue-900 text-blue-200",
		"plan":    "bg-indigo-900 text-indigo-200",
		"build":   "bg-yellow-900 text-yellow-200",
		"evaluate":"bg-purple-900 text-purple-200",
	}
	c := classes[phase]
	if c == "" {
		c = "bg-gray-700"
	}
	return template.HTML(fmt.Sprintf(`<span class="px-2 py-0.5 %s rounded text-xs">%s</span>`, c, phase))
}

func priorityBadge(p int) template.HTML {
	if p <= 1 {
		return `<span class="px-2 py-0.5 bg-red-900 text-red-200 rounded text-xs font-bold">P0-P1</span>`
	}
	if p <= 2 {
		return `<span class="px-2 py-0.5 bg-orange-900 text-orange-200 rounded text-xs">P2</span>`
	}
	return template.HTML(fmt.Sprintf(`<span class="px-2 py-0.5 bg-gray-700 rounded text-xs">P%d</span>`, p))
}

func labelTag(label string) template.HTML {
	if strings.HasPrefix(label, "sdp:") {
		return template.HTML(fmt.Sprintf(`<span class="px-2 py-0.5 bg-slate-700 rounded text-xs">⚙️ %s</span>`, label))
	}
	return template.HTML(fmt.Sprintf(`<span class="px-2 py-0.5 bg-gray-700 rounded text-xs">%s</span>`, label))
}

func strVal(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		return fmt.Sprint(int(val))
	default:
		return fmt.Sprint(val)
	}
}

func intVal(m map[string]any, key string) int {
	v, _ := m[key]
	switch val := v.(type) {
	case float64:
		return int(val)
	default:
		return 0
	}
}
