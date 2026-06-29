package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/services"
	"yuem-go/backend-gin/internal/storage"
)

func (h NativeHandlers) AdminDatabaseOverview(c *gin.Context) {
	if h.DB == nil {
		writeSuccess(c, matrixMsgOK, gin.H{"configured": false})
		return
	}
	dialect := ""
	if h.DB.Dialector != nil {
		dialect = h.DB.Dialector.Name()
	}
	payload := gin.H{"configured": true, "driver": dialect}
	switch dialect {
	case "postgres":
		var version string
		_ = h.DB.WithContext(c.Request.Context()).Raw(`SELECT version()`).Scan(&version).Error
		var dbName string
		_ = h.DB.WithContext(c.Request.Context()).Raw(`SELECT current_database()`).Scan(&dbName).Error
		var totalBytes int64
		_ = h.DB.WithContext(c.Request.Context()).Raw(`SELECT pg_database_size(current_database())`).Scan(&totalBytes).Error
		var tableCount int64
		_ = h.DB.WithContext(c.Request.Context()).Raw(`SELECT COUNT(1) FROM information_schema.tables WHERE table_schema = current_schema() AND table_type = 'BASE TABLE'`).Scan(&tableCount).Error
		payload["database"] = dbName
		payload["version"] = version
		payload["total_bytes"] = totalBytes
		payload["table_count"] = tableCount
	case "mysql":
		var version string
		_ = h.DB.WithContext(c.Request.Context()).Raw(`SELECT VERSION()`).Scan(&version).Error
		var dbName string
		_ = h.DB.WithContext(c.Request.Context()).Raw(`SELECT DATABASE()`).Scan(&dbName).Error
		var totalBytes int64
		_ = h.DB.WithContext(c.Request.Context()).Raw(`SELECT COALESCE(SUM(data_length + index_length), 0) FROM information_schema.tables WHERE table_schema = DATABASE()`).Scan(&totalBytes).Error
		var tableCount int64
		_ = h.DB.WithContext(c.Request.Context()).Raw(`SELECT COUNT(1) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_type = 'BASE TABLE'`).Scan(&tableCount).Error
		payload["database"] = dbName
		payload["version"] = version
		payload["total_bytes"] = totalBytes
		payload["table_count"] = tableCount
	default:
		payload["message"] = "database driver is not supported by the database manager"
	}
	writeSuccess(c, matrixMsgOK, payload)
}

func (h NativeHandlers) AdminDatabaseTables(c *gin.Context) {
	if h.DB == nil {
		writeSuccess(c, matrixMsgOK, gin.H{"items": []gin.H{}})
		return
	}
	keyword := strings.ToLower(strings.TrimSpace(c.Query("keyword")))
	items := []gin.H{}
	switch h.DB.Dialector.Name() {
	case "postgres":
		type row struct {
			Name       string `gorm:"column:name"`
			Rows       int64  `gorm:"column:rows"`
			TableBytes int64  `gorm:"column:table_bytes"`
			IndexBytes int64  `gorm:"column:index_bytes"`
			TotalBytes int64  `gorm:"column:total_bytes"`
		}
		var rows []row
		err := h.DB.WithContext(c.Request.Context()).Raw(`
			SELECT relname AS name,
				GREATEST(reltuples::bigint, 0) AS rows,
				pg_relation_size(c.oid) AS table_bytes,
				pg_indexes_size(c.oid) AS index_bytes,
				pg_total_relation_size(c.oid) AS total_bytes
			FROM pg_class c
			JOIN pg_namespace n ON n.oid = c.relnamespace
			WHERE c.relkind = 'r' AND n.nspname = current_schema()
			ORDER BY pg_total_relation_size(c.oid) DESC, relname ASC`).Scan(&rows).Error
		if writeDBError(c, err, "") {
			return
		}
		for _, row := range rows {
			if keyword != "" && !strings.Contains(strings.ToLower(row.Name), keyword) {
				continue
			}
			items = append(items, gin.H{"name": row.Name, "rows": row.Rows, "table_bytes": row.TableBytes, "index_bytes": row.IndexBytes, "total_bytes": row.TotalBytes})
		}
	case "mysql":
		type row struct {
			Name       string `gorm:"column:name"`
			Rows       int64  `gorm:"column:rows"`
			TableBytes int64  `gorm:"column:table_bytes"`
			IndexBytes int64  `gorm:"column:index_bytes"`
			TotalBytes int64  `gorm:"column:total_bytes"`
		}
		var rows []row
		err := h.DB.WithContext(c.Request.Context()).Raw(`
			SELECT table_name AS name,
				COALESCE(table_rows, 0) AS rows,
				COALESCE(data_length, 0) AS table_bytes,
				COALESCE(index_length, 0) AS index_bytes,
				COALESCE(data_length + index_length, 0) AS total_bytes
			FROM information_schema.tables
			WHERE table_schema = DATABASE() AND table_type = 'BASE TABLE'
			ORDER BY total_bytes DESC, table_name ASC`).Scan(&rows).Error
		if writeDBError(c, err, "") {
			return
		}
		for _, row := range rows {
			if keyword != "" && !strings.Contains(strings.ToLower(row.Name), keyword) {
				continue
			}
			items = append(items, gin.H{"name": row.Name, "rows": row.Rows, "table_bytes": row.TableBytes, "index_bytes": row.IndexBytes, "total_bytes": row.TotalBytes})
		}
	}
	writeSuccess(c, matrixMsgOK, gin.H{"items": items, "total": len(items)})
}

func (h NativeHandlers) AdminDatabaseColumns(c *gin.Context) {
	table := strings.TrimSpace(c.Param("table"))
	if h.DB == nil || table == "" {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "invalid table", nil)
		return
	}
	items := []gin.H{}
	switch h.DB.Dialector.Name() {
	case "postgres":
		type row struct {
			Name            string  `gorm:"column:name"`
			DataType        string  `gorm:"column:data_type"`
			Nullable        string  `gorm:"column:nullable"`
			DefaultValue    *string `gorm:"column:default_value"`
			OrdinalPosition int     `gorm:"column:ordinal_position"`
			AvgWidth        int64   `gorm:"column:avg_width"`
			EstimatedRows   int64   `gorm:"column:estimated_rows"`
		}
		var rows []row
		err := h.DB.WithContext(c.Request.Context()).Raw(`
			SELECT c.column_name AS name,
				c.data_type,
				c.is_nullable AS nullable,
				c.column_default AS default_value,
				c.ordinal_position,
				COALESCE(s.avg_width, 0) AS avg_width,
				COALESCE(pc.reltuples::bigint, 0) AS estimated_rows
			FROM information_schema.columns c
			LEFT JOIN pg_stats s ON s.schemaname = c.table_schema AND s.tablename = c.table_name AND s.attname = c.column_name
			LEFT JOIN pg_class pc ON pc.relname = c.table_name
			LEFT JOIN pg_namespace pn ON pn.oid = pc.relnamespace AND pn.nspname = c.table_schema
			WHERE c.table_schema = current_schema() AND c.table_name = ?
			ORDER BY c.ordinal_position ASC`, table).Scan(&rows).Error
		if writeDBError(c, err, "") {
			return
		}
		for _, row := range rows {
			items = append(items, gin.H{"name": row.Name, "data_type": row.DataType, "nullable": row.Nullable == "YES", "default": row.DefaultValue, "avg_width": row.AvgWidth, "estimated_bytes": row.AvgWidth * row.EstimatedRows, "ordinal_position": row.OrdinalPosition})
		}
	case "mysql":
		type row struct {
			Name            string  `gorm:"column:name"`
			DataType        string  `gorm:"column:data_type"`
			Nullable        string  `gorm:"column:nullable"`
			DefaultValue    *string `gorm:"column:default_value"`
			OrdinalPosition int     `gorm:"column:ordinal_position"`
		}
		var rows []row
		err := h.DB.WithContext(c.Request.Context()).Raw(`
			SELECT column_name AS name, column_type AS data_type, is_nullable AS nullable, column_default AS default_value, ordinal_position
			FROM information_schema.columns
			WHERE table_schema = DATABASE() AND table_name = ?
			ORDER BY ordinal_position ASC`, table).Scan(&rows).Error
		if writeDBError(c, err, "") {
			return
		}
		for _, row := range rows {
			items = append(items, gin.H{"name": row.Name, "data_type": row.DataType, "nullable": row.Nullable == "YES", "default": row.DefaultValue, "ordinal_position": row.OrdinalPosition})
		}
	}
	writeSuccess(c, matrixMsgOK, gin.H{"table": table, "items": items, "total": len(items)})
}

func (h NativeHandlers) AdminDatabaseIndexAudit(c *gin.Context) {
	writeSuccess(c, matrixMsgOK, gin.H{"issues": storage.AuditSchema(c.Request.Context(), h.DB), "desired_indexes": storage.DesiredIndexes()})
}

func (h NativeHandlers) AdminDatabaseRepair(c *gin.Context) {
	issues, err := storage.RepairSchema(c.Request.Context(), h.DB)
	if writeDBError(c, err, "") {
		return
	}
	writeSuccess(c, "database schema repair completed", gin.H{"issues": issues})
}

func (h NativeHandlers) AdminDatabaseVacuumConfig(c *gin.Context) {
	cfg := services.ReadDatabaseVacuumConfig(h.Settings)
	payload := gin.H{
		"configured": h.DB != nil,
		"supported":  h.DB != nil && h.DB.Dialector != nil && h.DB.Dialector.Name() == "postgres",
		"config":     cfg,
	}
	if payload["supported"] == true {
		tables, err := services.PostgresCurrentSchemaTables(c.Request.Context(), h.DB)
		if writeDBError(c, err, "") {
			return
		}
		payload["available_tables"] = tables
	} else {
		payload["available_tables"] = []string{}
		payload["message"] = "database vacuum analyze is only supported for PostgreSQL"
	}
	writeSuccess(c, matrixMsgOK, payload)
}

func (h NativeHandlers) AdminDatabaseVacuumConfigUpdate(c *gin.Context) {
	if h.Settings == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return
	}
	body := readBodyMap(c)
	enabled, _ := boolFromAny(body["enabled"])
	intervalHours, _ := intFromAny(body["interval_hours"])
	nextRunAt := strings.TrimSpace(toString(body["next_run_at"]))
	tables := stringSliceFromAny(body["tables"])
	if enabled {
		if _, err := services.ValidatePostgresVacuumTables(c.Request.Context(), h.DB, tables); err != nil {
			writeVacuumError(c, err)
			return
		}
	}
	if nextRunAt != "" {
		if _, err := time.Parse(time.RFC3339Nano, nextRunAt); err != nil {
			if _, fallbackErr := time.Parse(time.RFC3339, nextRunAt); fallbackErr != nil {
				response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "invalid next_run_at", nil)
				return
			}
		}
	}
	cfg := services.DatabaseVacuumConfig{
		Enabled:       enabled,
		Tables:        tables,
		IntervalHours: intervalHours,
		NextRunAt:     nextRunAt,
	}
	if !services.SaveDatabaseVacuumConfig(c.Request.Context(), h.Settings, cfg) {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return
	}
	h.AdminDatabaseVacuumConfig(c)
}

func (h NativeHandlers) AdminDatabaseVacuumAnalyze(c *gin.Context) {
	body := readBodyMap(c)
	tables := stringSliceFromAny(body["tables"])
	if table := strings.TrimSpace(toString(body["table"])); table != "" {
		tables = append(tables, table)
	}
	result, err := services.RunPostgresVacuumAnalyze(c.Request.Context(), h.DB, tables)
	if err != nil {
		writeVacuumError(c, err)
		return
	}
	cfg := services.ReadDatabaseVacuumConfig(h.Settings)
	next := time.Now().UTC().Add(time.Duration(cfg.IntervalHours) * time.Hour).Format(time.RFC3339Nano)
	_ = services.SaveDatabaseVacuumRun(c.Request.Context(), h.Settings, next, result)
	writeSuccess(c, "database vacuum analyze completed", gin.H{"result": result, "config": services.ReadDatabaseVacuumConfig(h.Settings)})
}

func writeVacuumError(c *gin.Context, err error) {
	if err == nil {
		return
	}
	if errors.Is(err, services.ErrUnsupportedVacuumDriver) {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, err.Error(), nil)
		return
	}
	response.JSON(c, http.StatusBadRequest, response.CodeValidationError, err.Error(), nil)
}
