package spa

import (
	"crypto/sha256"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v3"
)

//go:embed all:dist
var distFS embed.FS

var mimeTypes = map[string]string{
	".html":  "text/html; charset=utf-8",
	".css":   "text/css; charset=utf-8",
	".js":    "application/javascript",
	".json":  "application/json",
	".svg":   "image/svg+xml",
	".webp":  "image/webp",
	".png":   "image/png",
	".ico":   "image/x-icon",
	".woff":  "font/woff",
	".woff2": "font/woff2",
	".ttf":   "font/ttf",
}

func contentType(path string) string {
	ext := filepath.Ext(path)
	if ct, ok := mimeTypes[ext]; ok {
		return ct
	}
	return "application/octet-stream"
}

// Register mounts the SPA static file handler on the Fiber app.
// Must be called after all API routes are registered.
func Register(app *fiber.App) {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		log.Println("[spa] no web build found, skipping static file serving")
		return
	}

	index, err := fs.ReadFile(sub, "index.html")
	if err != nil {
		log.Println("[spa] index.html not found in build, skipping static file serving")
		return
	}

	hash := sha256.Sum256(index)
	indexETag := fmt.Sprintf(`"%x"`, hash[:8])

	log.Println("[spa] serving web panel from embedded build")

	app.Get("/*", func(c fiber.Ctx) error {
		path := strings.TrimPrefix(c.Path(), "/")
		if path == "" {
			path = "index.html"
		}

		data, err := fs.ReadFile(sub, path)
		if err != nil {
			if c.Get("If-None-Match") == indexETag {
				return c.SendStatus(304)
			}
			c.Set("Content-Type", "text/html; charset=utf-8")
			c.Set("Cache-Control", "no-cache")
			c.Set("ETag", indexETag)
			return c.Send(index)
		}

		c.Set("Content-Type", contentType(path))

		if path == "index.html" {
			if c.Get("If-None-Match") == indexETag {
				return c.SendStatus(304)
			}
			c.Set("Cache-Control", "no-cache")
			c.Set("ETag", indexETag)
		} else if strings.HasPrefix(path, "assets/") {
			c.Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			c.Set("Cache-Control", "public, max-age=86400")
		}

		return c.Send(data)
	})
}
