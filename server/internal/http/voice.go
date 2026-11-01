package httpapi

import (
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/buricodes/fuck-algo/server/internal/http/middleware"
	"github.com/buricodes/fuck-algo/server/internal/service/voice"
	"github.com/buricodes/fuck-algo/server/internal/store"
	"github.com/buricodes/fuck-algo/server/pkg/response"
)

var audioExtensions = map[string]string{
	"audio/webm": ".webm",
	"video/webm": ".webm",
	"audio/ogg":  ".ogg",
	".webm":      ".webm",
	".ogg":       ".ogg",
	".oga":       ".ogg",
}

type tokenResponse struct {
	Token string `json:"token"`
}

type transcriptResponse struct {
	Transcript string `json:"transcript"`
}

func (h *handler) voiceToken(c *gin.Context) {
	if !h.STT.Available() {
		response.Fail(c, unavailable("Voice"))
		return
	}
	ctx, u := c.Request.Context(), currentUser(c)
	if _, err := h.spend(ctx, u, store.CounterVoice, 1); err != nil {
		response.Fail(c, err)
		return
	}
	token, err := h.STT.RealtimeToken(ctx)
	if err != nil {
		h.refund(ctx, u, store.CounterVoice, 1)
		response.Fail(c, err)
		return
	}
	response.OK(c, tokenResponse{Token: token})
}

func (h *handler) voiceTranscribe(c *gin.Context) {
	if !h.STT.Available() {
		response.Fail(c, unavailable("Voice"))
		return
	}
	audio, filename, err := h.readAudio(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	text, err := h.STT.Transcribe(c.Request.Context(), audio, filename)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, transcriptResponse{Transcript: strings.TrimSpace(text)})
}

func (h *handler) readAudio(c *gin.Context) ([]byte, string, error) {
	header, err := c.FormFile("audio")
	var tooLarge *http.MaxBytesError
	switch {
	case errors.As(err, &tooLarge):
		return nil, "", middleware.TooLarge()
	case err != nil:
		return nil, "", badRequest("Send the recording as multipart form data in a field called audio.")
	case header.Size == 0 || header.Size > h.Config.Limits.AudioBytes:
		return nil, "", badRequest("The recording must be between 1 byte and 5 MB.")
	}
	ext, ok := audioExtension(header)
	if !ok {
		return nil, "", badRequest("The recording must be webm or ogg audio.")
	}
	audio, err := readPart(header)
	if err != nil {
		return nil, "", err
	}
	return audio, "audio" + ext, nil
}

func audioExtension(header *multipart.FileHeader) (string, bool) {
	if mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type")); err == nil {
		if ext, ok := audioExtensions[mediaType]; ok {
			return ext, true
		}
	}
	ext, ok := audioExtensions[strings.ToLower(filepath.Ext(header.Filename))]
	return ext, ok
}

func readPart(header *multipart.FileHeader) ([]byte, error) {
	f, err := header.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

func (h *handler) voiceRules(c *gin.Context) {
	if !h.Voice.Available() {
		response.Fail(c, unavailable("Voice rules"))
		return
	}
	var req voice.Request
	if err := bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	now, err := voice.Validate(req)
	if err != nil {
		response.Fail(c, badRequest(err.Error()))
		return
	}
	result, err := h.Voice.Rules(c.Request.Context(), req, now)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, result)
}
