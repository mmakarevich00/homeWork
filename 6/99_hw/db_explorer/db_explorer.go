package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type DBExplorer struct {
	db     *sql.DB
	tables map[string]Table
}
type Table struct {
	Name       string
	Columns    []Column
	PrimaryKey string
}
type Column struct {
	Name         string
	Type         string
	IsNullable   bool
	IsPrimaryKey bool
}

func NewDbExplorer(db *sql.DB) (http.Handler, error) {
	explorer := &DBExplorer{
		db:     db,
		tables: make(map[string]Table),
	}

	err := explorer.loadTable()
	if err != nil {
		return nil, err
	}
	return explorer, nil
}

func (dbe *DBExplorer) loadTable() error {
	rows, err := dbe.db.Query("SHOW TABLES")
	if err != nil {
		return err
	}

	var tableNames []string
	for rows.Next() {
		var name string
		rows.Scan(&name)
		tableNames = append(tableNames, name)
	}

	for _, tableName := range tableNames {
		table, err := dbe.getTableStructure(tableName)
		if err != nil {
			return err
		}
		dbe.tables[tableName] = table
	}
	return nil
}

func (dbe *DBExplorer) getTableStructure(tableName string) (Table, error) {
	table := Table{
		Name:    tableName,
		Columns: []Column{},
	}

	query := fmt.Sprintf("SHOW FULL COLUMNS FROM `%s`", tableName)
	rows, err := dbe.db.Query(query)
	if err != nil {
		return table, err
	}

	for rows.Next() {
		var field, colType, collation, null, key, defaultVal, extra, privileges, comment sql.NullString
		rows.Scan(&field, &colType, &collation, &null, &key, &defaultVal, &extra, &privileges, &comment)

		column := Column{
			Name:         field.String,
			Type:         colType.String,
			IsNullable:   null.String == "YES",
			IsPrimaryKey: key.String == "PRI",
		}
		table.Columns = append(table.Columns, column)

		if column.IsPrimaryKey {
			table.PrimaryKey = column.Name
		}
	}
	return table, nil
}

func (dbe *DBExplorer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	path := strings.Trim(r.URL.Path, "/")
	splitPath := strings.Split(path, "/")

	tableName := splitPath[0]

	switch r.Method {
	case "GET":
		if len(splitPath) == 1 {
			dbe.showData(w, r, tableName)
		} else {
			dbe.showDataById(w, r, tableName, splitPath[1])
		}
	case "PUT":
		dbe.createData(w, r, tableName)
	case "POST":
		dbe.updateData(w, r, tableName, splitPath[1])
	case "DELETE":
		dbe.deleteData(w, r, tableName, splitPath[1])
	}
}

func (dbe *DBExplorer) showTable(w http.ResponseWriter) {
	var tableNames []string
	for tableName := range dbe.tables {
		tableNames = append(tableNames, tableName)
	}

	var response = map[string]interface{}{
		"response": map[string]interface{}{
			"tables": tableNames,
		},
	}
	json.NewEncoder(w).Encode(response)
}

func (dbe *DBExplorer) sendError(w http.ResponseWriter, message string, code int) {
	w.WriteHeader(code)
	response := map[string]interface{}{
		"error": message,
	}
	json.NewEncoder(w).Encode(response)
}

func (dbe *DBExplorer) showData(w http.ResponseWriter, r *http.Request, tableName string) {
	limit := 5
	offset := 0

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil {
			offset = parsed
		}
	}

	query := fmt.Sprintf("SELECT * FROM `%s` LIMIT ? OFFSET ?", tableName)
	rows, err := dbe.db.Query(query, limit, offset)
	if err != nil {
		dbe.sendError(w, err.Error(), 500)
		return
	}

	records, err := dbe.readRows(rows)
	if err != nil {
		dbe.sendError(w, err.Error(), 500)
		return
	}

	var response = map[string]interface{}{
		"response": map[string]interface{}{
			"records": records,
		},
	}
	json.NewEncoder(w).Encode(response)
}

func (dbe *DBExplorer) showDataById(w http.ResponseWriter, r *http.Request, tableName, idStr string) {
	table := dbe.tables[tableName]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		dbe.sendError(w, "Invalid id", 400)
		return
	}

	query := fmt.Sprintf("SELECT * FROM `%s` WHERE `%s` = ?", tableName, table.PrimaryKey)
	rows, err := dbe.db.Query(query, id)
	if err != nil {
		dbe.sendError(w, err.Error(), 500)
		return
	}

	records, err := dbe.readRows(rows)
	if err != nil {
		dbe.sendError(w, err.Error(), 500)
		return
	}

	if len(records) == 0 {
		dbe.sendError(w, "record not found", 404)
		return
	}

	var response = map[string]interface{}{
		"response": map[string]interface{}{
			"record": records[0],
		},
	}
	json.NewEncoder(w).Encode(response)
}

func (dbe *DBExplorer) createData(w http.ResponseWriter, r *http.Request, tableName string) {
	table := dbe.tables[tableName]

	var data map[string]interface{}
	json.NewDecoder(r.Body).Decode(&data)

	var keys []string
	var values []interface{}

	for _, col := range table.Columns {
		if value, exists := data[col.Name]; exists {
			if value == nil && !col.IsNullable {
				dbe.sendError(w, fmt.Sprintf("field %s have invalid type", col.Name), 400)
				return
			}

			keys = append(keys, "`"+col.Name+"`")
			values = append(values, value)
		} else if !col.IsNullable {
			keys = append(keys, "`"+col.Name+"`")
			if strings.Contains(col.Type, "int") {
				values = append(values, 0)
			} else {
				values = append(values, "")
			}
		}
	}

	placeholders := strings.Repeat("?,", len(values))
	placeholders = placeholders[:len(placeholders)-1]
	query := fmt.Sprintf("INSERT INTO `%s` (%s) VALUES (%s)",
		tableName, strings.Join(keys, ","), placeholders)

	result, err := dbe.db.Exec(query, values...)
	if err != nil {
		dbe.sendError(w, err.Error(), 500)
		return
	}
	lastID, _ := result.LastInsertId()

	var response = map[string]interface{}{
		"response": map[string]interface{}{
			table.PrimaryKey: int(lastID),
		},
	}
	json.NewEncoder(w).Encode(response)
}

func (dbe *DBExplorer) updateData(w http.ResponseWriter, r *http.Request, tableName, idStr string) {
	table := dbe.tables[tableName]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		dbe.sendError(w, "invalid id", 400)
		return
	}

	var data map[string]interface{}
	json.NewDecoder(r.Body).Decode(&data)

	var setSplits []string
	var values []interface{}

	for fieldName, value := range data {
		var column *Column
		for _, col := range table.Columns {
			if col.Name == fieldName {
				column = &col
				break
			}
		}

		if column == nil {
			continue
		}

		if column.IsPrimaryKey {
			dbe.sendError(w, fmt.Sprintf("field %s have invalid type", fieldName), 400)
			return
		}

		if value == nil && !column.IsNullable {
			dbe.sendError(w, fmt.Sprintf("field %s have invalid type", fieldName), 400)
			return
		}

		setSplits = append(setSplits, fmt.Sprintf("`%s` = ?", fieldName))
		values = append(values, value)
	}

	if len(setSplits) == 0 {
		dbe.sendError(w, "nothing to update", 400)
		return
	}

	values = append(values, id)
	query := fmt.Sprintf("UPDATE `%s` SET %s WHERE `%s` = ?",
		tableName, strings.Join(setSplits, ", "), table.PrimaryKey)

	result, err := dbe.db.Exec(query, values...)
	if err != nil {
		dbe.sendError(w, err.Error(), 500)
		return
	}

	affected, _ := result.RowsAffected()

	var response = map[string]interface{}{
		"response": map[string]interface{}{
			"updated": int(affected),
		},
	}
	json.NewEncoder(w).Encode(response)
}

func (dbe *DBExplorer) deleteData(w http.ResponseWriter, r *http.Request, tableName, idStr string) {
	table := dbe.tables[tableName]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		dbe.sendError(w, "invalid id", 400)
		return
	}

	query := fmt.Sprintf("DELETE FROM `%s` WHERE `%s` = ?", tableName, table.PrimaryKey)
	result, err := dbe.db.Exec(query, id)
	if err != nil {
		dbe.sendError(w, err.Error(), 500)
		return
	}

	affected, _ := result.RowsAffected()

	var response = map[string]interface{}{
		"response": map[string]interface{}{
			"deleted": int(affected),
		},
	}
	json.NewEncoder(w).Encode(response)
}

func (dbe *DBExplorer) readRows(rows *sql.Rows) ([]map[string]interface{}, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}

	for rows.Next() {
		values := make([]interface{}, len(columns))
		pointers := make([]interface{}, len(columns))
		for i := range values {
			pointers[i] = &values[i]
		}

		err := rows.Scan(pointers...)
		if err != nil {
			return nil, err
		}

		data := make(map[string]interface{})
		for i, colName := range columns {
			val := values[i]
			if val == nil {
				data[colName] = nil
			} else {
				if bytes, ok := val.([]byte); ok {
					data[colName] = string(bytes)
				} else if intVal, ok := val.(int64); ok {
					data[colName] = int(intVal)
				} else {
					data[colName] = val
				}
			}
		}
		result = append(result, data)
	}

	return result, nil
}
