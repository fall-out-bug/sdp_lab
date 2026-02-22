package adapter

import (
	"context"
	"encoding/json"
	"strings"

	"sdp_dev/api/v1alpha1"
	"sdp_dev/internal/beads"
	"sdp_dev/internal/bus"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ReadySubscriber handles sdp.beads.*.ready events and creates Task CRDs.
type ReadySubscriber struct {
	Client         client.Client
	Namespace      string
	IntentTranslator *IntentTranslator
	PolicyGate     *PolicyGate
	LockManager    *RunLockManager
	BeadsAdapter   *beads.Adapter
}

// ReadyPayload is the envelope payload for sdp.beads.<project>.ready.
type ReadyPayload struct {
	ProjectID string        `json:"project_id"`
	Issues    []beads.Issue `json:"issues"`
	Count     int           `json:"count"`
}

// Handle processes a ready envelope: for each issue, translate → policy → lock → create Task.
func (r *ReadySubscriber) Handle(ctx context.Context, env bus.Envelope) {
	var payload ReadyPayload
	if len(env.Payload) > 0 {
		if err := json.Unmarshal(env.Payload, &payload); err != nil {
			log.FromContext(ctx).Error(err, "ready subscriber: parse payload")
			return
		}
	}
	if payload.ProjectID == "" {
		payload.ProjectID = env.ProjectID
	}
	if payload.ProjectID == "" {
		payload.ProjectID = "default"
	}

	for i := range payload.Issues {
		issue := &payload.Issues[i]
		if issue.ID == "" {
			continue
		}
		runID := issue.ID + "-1"
		if r.LockManager != nil {
			_, acquired, err := r.LockManager.TryAcquire(issue.ID, runID)
			if err != nil || !acquired {
				continue
			}
		}
		model := "glm-4.7"
		for _, l := range issue.Labels {
			if strings.HasPrefix(l, "model:") {
				model = strings.TrimPrefix(l, "model:")
				break
			}
		}
		if r.PolicyGate != nil {
			gr := r.PolicyGate.PreDispatchModelAllowlist(model)
			if !gr.Passed {
				continue
			}
		}
		intent, err := r.IntentTranslator.Translate(issue, runID)
		if err != nil {
			log.FromContext(ctx).Error(err, "ready subscriber: translate", "issue", issue.ID)
			continue
		}
		task := r.taskFromIntent(intent, model, payload.ProjectID)
		if err := r.Client.Create(ctx, task); err != nil {
			if !isAlreadyExists(err) {
				log.FromContext(ctx).Error(err, "ready subscriber: create Task", "issue", issue.ID)
			}
			continue
		}
		if r.BeadsAdapter != nil {
			_ = r.BeadsAdapter.Claim(issue.ID)
		}
		log.FromContext(ctx).Info("created Task from ready event", "issue", issue.ID, "task", task.Name)
	}
}

func (r *ReadySubscriber) taskFromIntent(intent *TaskIntent, model, projectID string) *v1alpha1.Task {
	name := dnsName(intent.IssueID)
	if len(name) > 63 {
		name = name[:63]
	}
	ns := r.Namespace
	if ns == "" {
		ns = "kubeopencode-system"
	}
	labels := map[string]string{
		"beads.issue": intent.IssueID,
		"sdp.run_id":  intent.RunID,
	}
	if projectID != "" {
		labels["sdp.project"] = projectID
	}
	for k, v := range intent.Labels {
		labels[k] = v
	}
	return &v1alpha1.Task{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Labels:    labels,
		},
		Spec: v1alpha1.TaskSpec{
			Prompt:    intent.Prompt,
			Objective: intent.Objective,
			AgentRef:  v1alpha1.AgentRef{Model: model},
		},
		Status: v1alpha1.TaskStatus{
			Phase: v1alpha1.TaskPhasePending,
		},
	}
}

func dnsName(issueID string) string {
	s := strings.ToLower(strings.ReplaceAll(issueID, "_", "-"))
	var b strings.Builder
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			b.WriteRune(c)
		}
	}
	out := b.String()
	if out == "" {
		return "task-" + issueID
	}
	return out
}

func isAlreadyExists(err error) bool {
	return errors.IsAlreadyExists(err)
}
