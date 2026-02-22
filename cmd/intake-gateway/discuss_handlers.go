package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"sdp_dev/internal/beads"
	"sdp_dev/internal/bus"
	"sdp_dev/internal/discuss"
	"sdp_dev/internal/federation"
	"sdp_dev/internal/registry"
)

// handleDiscussSubmit handles POST /api/v1/discuss
func handleDiscussSubmit(store *discuss.Store, analyzer discuss.Analyzer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		var req discuss.DiscussRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if req.Title == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "title required"})
			return
		}

		sess, err := store.Create(req)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		if analyzer != nil {
			sess.Phase = discuss.PhaseAnalyzing
			_ = store.Update(sess)

			analysis, err := analyzer.Analyze(r.Context(), sess)
			if err != nil {
				sess.Phase = discuss.PhaseFailed
				sess.Error = err.Error()
				_ = store.Update(sess)
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "analysis failed: " + err.Error()})
				return
			}
			sess.Analysis = analysis
			sess.Phase = discuss.PhaseReady
			_ = store.Update(sess)
		}

		writeJSON(w, http.StatusCreated, map[string]any{
			"id":      sess.ID,
			"phase":   sess.Phase,
			"project": sess.ProjectID,
		})
	}
}

// handleDiscussStatus handles GET /api/v1/discuss/{id}
func handleDiscussStatus(store *discuss.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		sess, ok := store.Get(id)
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "session not found"})
			return
		}
		writeJSON(w, http.StatusOK, sess)
	}
}

// handleDiscussApprove handles POST /api/v1/discuss/{id}/approve.
// Publishes to NATS sdp.intake.{projectID} for Bridge to create issues; falls back to direct Beads when NATS unavailable.
func handleDiscussApprove(store *discuss.Store, b bus.Bus, reg *registry.Store, workDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		sess, ok := store.Get(id)
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "session not found"})
			return
		}
		if sess.Phase != discuss.PhaseReady {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "session not ready for approval"})
			return
		}
		if sess.Analysis == nil || len(sess.Analysis.Subtasks) == 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "no subtasks to create"})
			return
		}

		projectID := sess.ProjectID
		if projectID == "" {
			projects := reg.List()
			if len(projects) > 0 {
				projectID = projects[0].ID
			}
		}
		if projectID == "" {
			projectID = "sdp_dev"
		}

		if b != nil {
			// Publish batch intake to NATS; Bridge will create feature + subtasks
			batch := federation.IntakeBatchPayload{
				ProjectID: projectID,
				Feature: federation.IntakeBatchItem{
					Title:       sess.Title,
					Description: sess.Description,
				},
				Subtasks: make([]federation.IntakeBatchItem, 0, len(sess.Analysis.Subtasks)),
				DepEdges: make([]federation.IntakeDepEdge, 0),
			}
			for _, st := range sess.Analysis.Subtasks {
				batch.Subtasks = append(batch.Subtasks, federation.IntakeBatchItem{
					Title:       st.Title,
					Description: st.Description,
					Acceptance:  st.Acceptance,
				})
			}
			for i, st := range sess.Analysis.Subtasks {
				if st.DependsOnIndex >= 0 && st.DependsOnIndex < len(sess.Analysis.Subtasks) && st.DependsOnIndex != i {
					batch.DepEdges = append(batch.DepEdges, federation.IntakeDepEdge{Blocked: i, Blocker: st.DependsOnIndex})
				}
			}

			payload, _ := json.Marshal(batch)
			subject := "sdp.intake." + projectID
			env := bus.Envelope{
				IssueID:       id,
				ArtifactID:    "discuss-approve",
				ArtifactClass: "intake",
				Phase:         "approved",
				Role:          "gateway",
				Payload:       payload,
				ProjectID:     projectID,
			}
			if err := b.Publish(subject, env); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "publish intake: " + err.Error()})
				return
			}

			sess.Phase = discuss.PhaseApproved
			sess.CreatedIssueIDs = []string{"queued"} // Bridge will create; IDs not known yet
			_ = store.Update(sess)

			writeJSON(w, http.StatusOK, map[string]any{
				"id":      sess.ID,
				"phase":   sess.Phase,
				"message": "published to NATS; Bridge will create issues",
			})
			return
		}

		// Fallback: direct Beads when NATS unavailable
		if workDir == "" {
			workDir, _ = os.Getwd()
		}
		createdIDs, featureID, err := createDiscussIssuesDirect(workDir, sess)
		if err != nil {
			sess.Phase = discuss.PhaseFailed
			sess.Error = err.Error()
			_ = store.Update(sess)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		sess.Phase = discuss.PhaseApproved
		sess.CreatedIssueIDs = append([]string{featureID}, createdIDs...)
		_ = store.Update(sess)

		writeJSON(w, http.StatusOK, map[string]any{
			"id":             sess.ID,
			"phase":          sess.Phase,
			"feature_id":     featureID,
			"subtask_ids":    createdIDs,
			"created_issues": sess.CreatedIssueIDs,
		})
	}
}

func createDiscussIssuesDirect(workDir string, sess *discuss.Session) (createdIDs []string, featureID string, err error) {
	adapter := beads.NewAdapter(workDir)
	labels := []string{"autonomy", "strict-evidence", "workstream:builder", "lane:commit"}

	featureID, err = adapter.Create(beads.CreateOpts{
		Title:       sess.Title,
		Type:        "feature",
		Priority:    2,
		Description: sess.Description,
		Labels:      labels,
	})
	if err != nil {
		return nil, "", fmt.Errorf("create feature: %w", err)
	}

	createdIDs = make([]string, 0, len(sess.Analysis.Subtasks))
	for _, st := range sess.Analysis.Subtasks {
		taskID, err := adapter.Create(beads.CreateOpts{
			Title:       st.Title,
			Type:        "task",
			Priority:    2,
			Description: st.Description,
			Acceptance:  st.Acceptance,
			Labels:      labels,
			ParentID:    featureID,
		})
		if err != nil {
			return nil, "", fmt.Errorf("create subtask: %w", err)
		}
		createdIDs = append(createdIDs, taskID)
	}

	for i, st := range sess.Analysis.Subtasks {
		if st.DependsOnIndex >= 0 && st.DependsOnIndex < len(createdIDs) && st.DependsOnIndex != i {
			_ = adapter.DepAdd(createdIDs[i], createdIDs[st.DependsOnIndex])
		}
	}

	_ = adapter.Sync(false)
	return createdIDs, featureID, nil
}
