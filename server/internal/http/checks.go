package httpapi

import (
	"github.com/gin-gonic/gin"

	"github.com/buricodes/fuck-algo/server/internal/domain"
	"github.com/buricodes/fuck-algo/server/internal/service/checks"
	"github.com/buricodes/fuck-algo/server/internal/store"
	"github.com/buricodes/fuck-algo/server/pkg/response"
)

type checksResponse struct {
	Results []domain.CheckResult `json:"results"`
}

func (h *handler) runChecks(c *gin.Context) {
	if !h.Checks.Available() {
		response.Fail(c, unavailable("AI checks"))
		return
	}
	var req checks.Request
	if err := bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := checks.Validate(req); err != nil {
		response.Fail(c, badRequest(err.Error()))
		return
	}
	plan := h.Checks.Plan(req)
	if err := h.judge(c, plan); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, checksResponse{Results: ordered(req.Items, plan.Results)})
}

func (h *handler) judge(c *gin.Context, plan *checks.Plan) error {
	if len(plan.Pending) == 0 {
		return nil
	}
	ctx, in := c.Request.Context(), install(c)
	granted, err := h.spend(ctx, in, store.CounterChecks, len(plan.Pending))
	if err != nil {
		return err
	}
	judged, err := h.Checks.Judge(ctx, plan, plan.Pending[:granted])
	h.refund(ctx, in, store.CounterChecks, granted-judged)
	return err
}

func ordered(items []domain.CheckItem, results map[string]domain.CheckResult) []domain.CheckResult {
	out := make([]domain.CheckResult, 0, len(results))
	for _, item := range items {
		if r, ok := results[item.ID]; ok {
			out = append(out, r)
		}
	}
	return out
}
