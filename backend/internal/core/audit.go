package core

import "log"

// AuditEntry is one recorded change. Source is "api" or "mcp".
type AuditEntry struct {
	ID     int64  `json:"id"`
	Time   string `json:"time"`
	Source string `json:"source"`
	Action string `json:"action"`
	Target string `json:"target"`
	Detail string `json:"detail"`
}

// Audit records who did what. It never fails the caller.
func (s *Service) Audit(source, action, target, detail string) {
	if len(detail) > 2000 {
		detail = detail[:2000] + "..."
	}
	if _, err := s.DB.Exec(`INSERT INTO audit_log(ts,source,action,target,detail) VALUES(?,?,?,?,?)`,
		nowISO(), source, action, target, detail); err != nil {
		log.Printf("audit: %v", err)
	}
}

// ListAudit returns the most recent entries, newest first.
func (s *Service) ListAudit(limit int) ([]AuditEntry, error) {
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	rows, err := s.DB.Query(`SELECT id,ts,source,action,target,detail FROM audit_log ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AuditEntry{}
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(&e.ID, &e.Time, &e.Source, &e.Action, &e.Target, &e.Detail); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
