package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"

	"github.com/zhou1h/3xui-network-panel-v2/ent/job"
	"github.com/zhou1h/3xui-network-panel-v2/internal/app"
	"github.com/zhou1h/3xui-network-panel-v2/internal/xui"
)

const (
	TypeResourceCheck = "resource:health"
	TypeRealityScan   = "resource:reality-scan"
)

type Payload struct {
	JobID      int    `json:"jobId"`
	ResourceID int    `json:"resourceId"`
	Targets    string `json:"targets,omitempty"`
}

func New(taskType string, payload Payload) (*asynq.Task, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(taskType, data, asynq.MaxRetry(2), asynq.Timeout(2*time.Minute)), nil
}

type Handler struct{ App *app.App }

func (h *Handler) Register(mux *asynq.ServeMux) {
	mux.HandleFunc(TypeResourceCheck, h.checkResource)
	mux.HandleFunc(TypeRealityScan, h.scanReality)
}

func (h *Handler) begin(ctx context.Context, p Payload) error {
	now := time.Now()
	return h.App.DB.Job.UpdateOneID(p.JobID).SetStatus("running").SetProgress(10).SetStartedAt(now).Exec(ctx)
}
func (h *Handler) fail(ctx context.Context, p Payload, err error) error {
	now := time.Now()
	_ = h.App.DB.Job.UpdateOneID(p.JobID).SetStatus("failed").SetProgress(100).SetMessage(err.Error()).SetFinishedAt(now).Exec(ctx)
	_ = h.App.DB.Resource.UpdateOneID(p.ResourceID).SetStatus("error").SetUpdatedAt(now).Exec(ctx)
	return fmt.Errorf("%w: %v", asynq.SkipRetry, err)
}
func decodePayload(task *asynq.Task) (Payload, error) {
	var p Payload
	err := json.Unmarshal(task.Payload(), &p)
	if err != nil || p.JobID < 1 || p.ResourceID < 1 {
		return p, fmt.Errorf("invalid task payload")
	}
	return p, nil
}

func (h *Handler) client(ctx context.Context, p Payload) (*xui.Client, error) {
	resource, err := h.App.DB.Resource.Get(ctx, p.ResourceID)
	if err != nil {
		return nil, err
	}
	token, err := h.App.Cipher.Decrypt(resource.AccessTokenCiphertext)
	if err != nil {
		return nil, err
	}
	return xui.New(resource.ManagementURL, token)
}

func (h *Handler) checkResource(ctx context.Context, task *asynq.Task) error {
	p, err := decodePayload(task)
	if err != nil {
		return err
	}
	if err = h.begin(ctx, p); err != nil {
		return err
	}
	client, err := h.client(ctx, p)
	if err != nil {
		return h.fail(ctx, p, err)
	}
	inv, err := client.Inventory(ctx)
	if err != nil {
		return h.fail(ctx, p, err)
	}
	now := time.Now()
	h.App.DB.Resource.UpdateOneID(p.ResourceID).SetStatus("ok").SetNodeCount(inv.Nodes).SetClientCount(inv.Clients).SetSocksCount(inv.Socks5).SetLatencyMs(inv.LatencyMS).SetLastCheckedAt(now).SaveX(ctx)
	payload := map[string]any{"inventory": inv}
	return h.App.DB.Job.UpdateOneID(p.JobID).SetStatus("completed").SetProgress(100).SetMessage("资源检测完成").SetPayload(payload).SetFinishedAt(now).Exec(ctx)
}

func (h *Handler) scanReality(ctx context.Context, task *asynq.Task) error {
	p, err := decodePayload(task)
	if err != nil {
		return err
	}
	if err = h.begin(ctx, p); err != nil {
		return err
	}
	client, err := h.client(ctx, p)
	if err != nil {
		return h.fail(ctx, p, err)
	}
	targets, err := client.ScanRealityTargets(ctx, p.Targets)
	if err != nil {
		return h.fail(ctx, p, err)
	}
	now := time.Now()
	message := "未找到可用 Reality 目标"
	if len(targets) > 0 {
		message = fmt.Sprintf("找到 %d 个可用目标，最优：%s", len(targets), targets[0].SNI)
	}
	return h.App.DB.Job.UpdateOneID(p.JobID).SetStatus("completed").SetProgress(100).SetMessage(message).SetPayload(map[string]any{"targets": targets}).SetFinishedAt(now).Exec(ctx)
}

var _ = job.StatusEQ
