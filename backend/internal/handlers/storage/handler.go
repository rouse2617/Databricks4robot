package storage

import (
	"context"
	"net/http"
	"os"
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
	// Register default resolvers on first init.
	initResolvers()
	return &Handler{gcsClient: client}, nil
}

func initResolvers() {
	if _, ok := GetResolver("grace"); ok {
		return // already registered
	}
	RegisterResolver("grace", &GraceResolver{
		devURL:       os.Getenv("GRACE_DEV_URL"),
		devUsername:  os.Getenv("GRACE_DEV_USERNAME"),
		devPassword:  os.Getenv("GRACE_DEV_PASSWORD"),
		prodURL:      os.Getenv("GRACE_PROD_URL"),
		prodUsername: os.Getenv("GRACE_PROD_USERNAME"),
		prodPassword: os.Getenv("GRACE_PROD_PASSWORD"),
	})
}

// --------------------------------------------------------------------------
// POST /api/v1/storage/resolve — resolve source ID → signed GCS URL
// --------------------------------------------------------------------------

type resolveRequest struct {
	Source string `json:"source" binding:"required"` // "grace"
	ID     string `json:"id" binding:"required"`     // video_id, asset_id, etc.
	Env    string `json:"env"`                       // "dev" or "prod", default "dev"
	Method string `json:"method"`                    // GET or PUT
	TTL    int    `json:"ttl"`
}

func (h *Handler) Resolve(c *gin.Context) {
	var req resolveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request", map[string]any{"error": err.Error()})
		return
	}

	env := strings.TrimSpace(req.Env)
	if env == "" {
		env = "dev"
	}

	resolver, ok := GetResolver(req.Source)
	if !ok {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "unknown source: "+req.Source, nil)
		return
	}

	gcsPath, err := resolver.Resolve(c.Request.Context(), req.ID, env)
	if err != nil {
		httpresp.Internal(c, "resolve failed: "+err.Error())
		return
	}

	bucket, object, ok := strings.Cut(gcsPath, "/")
	if !ok {
		httpresp.Internal(c, "invalid resolved path: "+gcsPath)
		return
	}

	method := strings.ToUpper(strings.TrimSpace(req.Method))
	if method == "" {
		method = "GET"
	}

	ttl := req.TTL
	if ttl <= 0 {
		ttl = 900
	}
	if ttl > 86400 {
		ttl = 86400
	}

	signedURL, err := storage.SignedURL(bucket, object, &storage.SignedURLOptions{
		Method:  method,
		Expires: time.Now().Add(time.Duration(ttl) * time.Second),
	})
	if err != nil {
		httpresp.Internal(c, "failed to sign URL: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"url":       signedURL,
		"method":    method,
		"expiresIn": ttl,
		"bucket":    bucket,
		"object":    object,
		"source":    req.Source,
		"env":       env,
	})
}

// --------------------------------------------------------------------------
// POST /api/v1/storage/sign-url — sign an arbitrary GCS URL
// --------------------------------------------------------------------------

type signURLRequest struct {
	Bucket string `json:"bucket" binding:"required"`
	Object string `json:"object" binding:"required"`
	Method string `json:"method"`
	TTL    int    `json:"ttl"`
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
