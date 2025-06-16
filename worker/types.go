package worker

import (
	"time"
)

// Statuts possibles d’une requête
type ReportStatus string

const (
	StatusWaiting    ReportStatus = "waiting"
	StatusProcessing ReportStatus = "processing"
	StatusComplete   ReportStatus = "complete"
	StatusError      ReportStatus = "error"
)

// Stockage d’une requête à traiter
type ReportRequest struct {
	ID         string                 // id unique
	Payload    map[string]interface{} // le json reçu du client
	Owner      string                 // user à l'origine
	Admin      bool                   // user admin ?
	Datasource string                 // ex: myreport
	CreatedAt  time.Time
}

// Résultat traité
type ReportResult struct {
	Status   ReportStatus
	Result   interface{} // []map[string]interface{} ou autre
	CSVPath  string
	ErrorMsg string
}
