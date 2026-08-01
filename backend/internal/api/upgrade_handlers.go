package api

import (
	"net/http"
	"strconv"

	"github.com/morphatrix/campmenu/internal/migrations"
	"github.com/morphatrix/campmenu/internal/models"
	"gorm.io/gorm"
)

const schemaVersionKey = "SCHEMA_VERSION"

func getSchemaVersion(db *gorm.DB) int {
	var row models.AppSetting
	if err := db.Where("key = ?", schemaVersionKey).First(&row).Error; err != nil {
		return 1
	}
	v, err := strconv.Atoi(row.Value)
	if err != nil {
		return 1
	}
	return v
}

func setSchemaVersion(db *gorm.DB, version int) error {
	var row models.AppSetting
	return db.Where(models.AppSetting{Key: schemaVersionKey}).
		Assign(models.AppSetting{Value: strconv.Itoa(version)}).
		FirstOrCreate(&row).Error
}

type pendingMigration struct {
	Version     int    `json:"version"`
	Description string `json:"description"`
}

type upgradeStatusResp struct {
	CurrentVersion int                `json:"currentVersion"`
	Pending        []pendingMigration `json:"pending"`
}

func (s *Server) handleUpgradeStatus(w http.ResponseWriter, _ *http.Request) {
	current := getSchemaVersion(s.DB)
	resp := upgradeStatusResp{CurrentVersion: current, Pending: []pendingMigration{}}
	for _, m := range migrations.Pending(current) {
		resp.Pending = append(resp.Pending, pendingMigration{Version: m.Version, Description: m.Description})
	}
	writeJSON(w, http.StatusOK, resp)
}

type upgradeVersionReq struct {
	Version int `json:"version"`
}

type upgradeResultResp struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

// nextMigrationFor validates that the requested version is exactly the one
// immediately following the current schema version — migrations must apply
// strictly in order, never skipped or out of sequence.
func nextMigrationFor(db *gorm.DB, version int) (migrations.Migration, string) {
	current := getSchemaVersion(db)
	if version != current+1 {
		return migrations.Migration{}, "cette migration doit être appliquée dans l'ordre (version attendue : " + strconv.Itoa(current+1) + ")"
	}
	m, ok := migrations.ByVersion(version)
	if !ok {
		return migrations.Migration{}, "migration introuvable"
	}
	return m, ""
}

// handleDryRunUpgrade runs the migration's statements inside a transaction
// that is always rolled back, to verify it applies cleanly without touching
// the database.
func (s *Server) handleDryRunUpgrade(w http.ResponseWriter, r *http.Request) {
	var req upgradeVersionReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps de requête invalide")
		return
	}
	m, errMsg := nextMigrationFor(s.DB, req.Version)
	if errMsg != "" {
		writeJSON(w, http.StatusOK, upgradeResultResp{OK: false, Error: errMsg})
		return
	}
	tx := s.DB.Begin()
	var execErr error
	for _, stmt := range m.Statements() {
		if err := tx.Exec(stmt).Error; err != nil {
			execErr = err
			break
		}
	}
	tx.Rollback()
	if execErr != nil {
		writeJSON(w, http.StatusOK, upgradeResultResp{OK: false, Error: execErr.Error()})
		return
	}
	writeJSON(w, http.StatusOK, upgradeResultResp{OK: true})
}

// handleApplyUpgrade runs the migration for real and bumps the stored schema
// version, atomically (both succeed or both roll back).
func (s *Server) handleApplyUpgrade(w http.ResponseWriter, r *http.Request) {
	var req upgradeVersionReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps de requête invalide")
		return
	}
	m, errMsg := nextMigrationFor(s.DB, req.Version)
	if errMsg != "" {
		writeJSON(w, http.StatusOK, upgradeResultResp{OK: false, Error: errMsg})
		return
	}
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		for _, stmt := range m.Statements() {
			if err := tx.Exec(stmt).Error; err != nil {
				return err
			}
		}
		return setSchemaVersion(tx, m.Version)
	})
	if err != nil {
		writeJSON(w, http.StatusOK, upgradeResultResp{OK: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, upgradeResultResp{OK: true})
}
