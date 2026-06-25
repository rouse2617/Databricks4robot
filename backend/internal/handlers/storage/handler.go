package storage

import (
	"context"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/option"

	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
)

type Handler struct {
	gcsClient *storage.Client
}

func NewHandler(ctx context.Context) (*Handler, error) {
	client, err := storage.NewClient(ctx, option.WithUserAgent("cyber-databrew-backend"))
	if err != nil {
		return nil, err
	}
	return &Handler{gcsClient: client}, nil
}

type signURLRequest struct {
	Bucket string `json:"bucket" binding:"required"`
	Object string `json:"object" binding:"required"`
	Method string `json:"method"` // GET or PUT, default GET
	TTL    int    `json:"ttl"`    // seconds, default 900 (15 min)
}

func (h *Handler) SignURL(c *gin.Context) {
	var req signURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request", map[string]any{"error": err.Error()})
		return
	}

	method := strings.ToUpper(strings.TrimSpace(req.Method))
	if method == "" {
		method = "GET"
	}
	if method != "GET" && method != "PUT" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "method must be GET or PUT", nil)
		return
	}

	ttl := req.TTL
	if ttl <= 0 {
		ttl = 900
	}
	if ttl > 86400 {
		ttl = 86400
	}

	url, err := storage.SignedURL(req.Bucket, req.Object, &storage.SignedURLOptions{
		Method:  method,
		Expires: time.Now().Add(time.Duration(ttl) * time.Second),
	})
	if err != nil {
		httpresp.Internal(c, "failed to sign URL: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"url":       url,
		"method":    method,
		"expiresIn": ttl,
		"bucket":    req.Bucket,
		"object":    req.Object,
	})
}
