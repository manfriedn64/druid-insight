package static

import (
	"druid-insight/auth"
	"druid-insight/logging"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Sert les fichiers statiques avec whitelist et fallback (admin/static_default)
func RegisterStaticHandler(cfg *auth.Config, accessLogger *logging.Logger) {
	staticDir := cfg.Server.Static
	if staticDir == "" {
		staticDir = "./static"
	}
	staticDefault := cfg.Server.StaticDefault
	if staticDefault == "" {
		staticDefault = "./static"
	}
	allowed := cfg.Server.StaticAllowed

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		reqPath := strings.TrimPrefix(r.URL.Path, "/")
		if reqPath == "" {
			reqPath = "index.html"
		}

		// Whitelist (wildcard support)
		if !isAllowedWildcard(reqPath, allowed) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			accessLogger.Write("[STATIC_REFUSED] " + reqPath)
			return
		}

		// Try static dir (admin)
		filePath := filepath.Join(staticDir, reqPath)
		content, err := os.ReadFile(filePath)
		if err == nil {
			final := applyTemplateMacros(string(content), cfg.Server.TemplateVars)
			w.Header().Set("Content-Type", mime.TypeByExtension(filepath.Ext(filePath)))
			w.Write([]byte(final))
			accessLogger.Write("[STATIC_OK] " + reqPath + " (DEFAULT)")
			return
		}

		// Fallback: static_default
		fallbackPath := filepath.Join(staticDefault, reqPath)
		content, err = os.ReadFile(fallbackPath)
		if err == nil {
			final := applyTemplateMacros(string(content), cfg.Server.TemplateVars)
			w.Header().Set("Content-Type", mime.TypeByExtension(filepath.Ext(fallbackPath)))
			w.Write([]byte(final))
			accessLogger.Write("[STATIC_OK] " + reqPath + " (DEFAULT)")
			return
		}

		http.NotFound(w, r)
		accessLogger.Write("[STATIC_NOTFOUND] " + reqPath)
	})
}

func applyTemplateMacros(content string, vars map[string]string) string {
	for key, val := range vars {
		placeholder := "{" + key + "}"
		content = strings.ReplaceAll(content, placeholder, val)
	}
	return content
}

// Vérifie si un nom de fichier est dans la whitelist (wildcard)
func isAllowedWildcard(fileName string, allowed []string) bool {
	for _, pattern := range allowed {
		// Normalise les slash pour compatibilité
		if matched, _ := filepath.Match(pattern, fileName); matched {
			return true
		}
		// Support backward compatibility for patterns like "*.js" in subfolders
		if strings.HasPrefix(pattern, "*/") {
			suffix := pattern[2:]
			if strings.HasSuffix(fileName, suffix) {
				return true
			}
		}
	}
	return false
}

// Vérifie si un fichier existe (et n'est pas un répertoire)
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	return err == nil && !info.IsDir()
}
