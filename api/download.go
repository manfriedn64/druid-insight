package api

import (
	"druid-insight/auth"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// DownloadReportCSV télécharge le CSV du rapport demandé (nécessite JWT valide)
func DownloadReportCSV(cfg *auth.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Validation du JWT
		username, _, err := auth.ExtractUserAndAdminFromJWT(r, cfg.JWT.Secret)
		if err != nil {
			http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
			return
		}

		// Extraction du paramètre GET id
		reportID := r.URL.Query().Get("id")
		if reportID == "" {
			http.Error(w, "Paramètre id manquant", http.StatusBadRequest)
			return
		}

		// Chemin du fichier
		csvPath := filepath.Join("csv", reportID+".csv")

		// Vérification existence
		if _, err := os.Stat(csvPath); err != nil {
			http.Error(w, "Fichier CSV non trouvé pour ce rapport", http.StatusNotFound)
			return
		}

		// Log (optionnel)
		fmt.Printf("[DOWNLOAD] user=%s id=%s path=%s\n", username, reportID, csvPath)

		// Envoi du fichier CSV
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"report_%s.csv\"", strings.ReplaceAll(reportID, "\"", "")))
		http.ServeFile(w, r, csvPath)
	}
}
