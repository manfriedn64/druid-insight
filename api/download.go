package api

import (
	"druid-insight/auth"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// DownloadReportCSV télécharge le CSV ou l'Excel du rapport demandé (nécessite JWT valide)
// Paramètre GET: id (obligatoire), type=csv|excel (optionnel, défaut: csv)
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

		// Extraction du type de fichier (csv par défaut)
		fileType := r.URL.Query().Get("type")
		if fileType == "" {
			fileType = "csv"
		}

		var filePath, contentType, fileName string
		switch strings.ToLower(fileType) {
		case "excel", "xlsx":
			filePath = filepath.Join("xls", reportID+".xlsx")
			contentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
			fileName = fmt.Sprintf("report_%s.xlsx", strings.ReplaceAll(reportID, "\"", ""))
		default:
			filePath = filepath.Join("csv", reportID+".csv")
			contentType = "text/csv"
			fileName = fmt.Sprintf("report_%s.csv", strings.ReplaceAll(reportID, "\"", ""))
		}

		// Vérification existence
		if _, err := os.Stat(filePath); err != nil {
			http.Error(w, "Fichier non trouvé pour ce rapport", http.StatusNotFound)
			return
		}

		// Log (optionnel)
		log.Printf("[DOWNLOAD] user=%s id=%s type=%s path=%s\n", username, reportID, fileType, filePath)

		// Envoi du fichier
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
		http.ServeFile(w, r, filePath)
	}
}
