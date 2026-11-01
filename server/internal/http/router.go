package httpapi

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/buricodes/fuck-algo/server/internal/http/middleware"
	"github.com/buricodes/fuck-algo/server/pkg/apperr"
	"github.com/buricodes/fuck-algo/server/pkg/response"
)

const (
	transcribeRoute = "/v1/voice/transcribe"
	healthTimeout   = 2 * time.Second
)

type healthResponse struct {
	Status string `json:"status"`
	DB     string `json:"db"`
}

func NewRouter(d Deps) (*gin.Engine, error) {
	h := &handler{Deps: d}
	r := gin.New()
	if err := r.SetTrustedProxies(d.Config.Server.TrustedProxies); err != nil {
		return nil, err
	}
	r.Use(middleware.Recovery(), middleware.RequestLogger(), middleware.CORS(middleware.CORSOptions{
		WebsiteURL:        d.Config.WebsiteURL,
		ExtensionOrigins:  d.Config.ExtensionOrigins,
		AnyExtensionInDev: d.Config.IsDevelopment(),
	}), middleware.BodyLimit(h.bodyLimit))
	r.NoRoute(func(c *gin.Context) { response.Fail(c, apperr.New(apperr.NotFound, "No such route.")) })

	r.GET("/healthz", h.health)
	h.routes(r.Group("/v1"))
	return r, nil
}

func (h *handler) routes(v1 *gin.RouterGroup) {
	limits := h.Config.RateLimits

	v1.POST("/webhooks/dodo", h.dodoWebhook)

	authed := v1.Group("", h.requireUser)
	authed.GET("/me", h.me)
	authed.POST("/checkout", middleware.RateLimit(limits.Checkout, byUser), h.createCheckout)
	authed.GET("/checkout/:sessionId", h.checkoutStatus)
	authed.POST("/voice/token", middleware.RateLimit(limits.VoiceToken, byUser), h.voiceToken)
	authed.POST("/voice/transcribe", middleware.RateLimit(limits.VoiceTranscribe, byUser), h.voiceTranscribe)
	authed.POST("/voice/rules", middleware.RateLimit(limits.VoiceRules, byUser), h.voiceRules)
	authed.POST("/checks", middleware.RateLimit(limits.Checks, byUser), h.runChecks)
}

func (h *handler) bodyLimit(route string) int64 {
	if route == transcribeRoute {
		return h.Config.Limits.TranscribeBodyBytes
	}
	return h.Config.Limits.BodyBytes
}

func (h *handler) health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), healthTimeout)
	defer cancel()
	db := "ok"
	if err := h.Store.Ping(ctx); err != nil {
		db = "down"
	}
	response.OK(c, healthResponse{Status: "ok", DB: db})
}
