package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	_ "os"

	_ "github.com/go-sql-driver/mysql"
)

type Command struct {
	Action string            `json:"action"`
	Table  string            `json:"table"`
	Data   map[string]string `json:"data,omitempty"`
	Query  map[string]string `json:"query,omitempty"`
	Attrs  []string          `json:"attrs,omitempty"`
	DBName string            `json:"dbname,omitempty"`
}
// Replace with actual IP Addresses of each slave device
var slaves = []string{
    "http://IP of slave 1:5001/slave", //Slave 1
	"http://IP of slave 2:5001/slave", //Slave 2
}


var db *sql.DB

func connectDatabase() {
	var err error
	dsn := "root:password of master device@tcp(localhost:3306)/" // Replace with actual MySQL credentials
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("Database connection failed:", err)
	}
	log.Println("Connected to MySQL server")
}

func handleCommand(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Master-Key")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var cmd Command
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		http.Error(w, "Invalid command", http.StatusBadRequest)
		return
	}
	log.Printf("Received command: %+v", cmd)

	switch cmd.Action {
	case "ping":
		respond(w, nil, "Slave connected successfully")

	case "create_db", "create_table", "drop_db", "drop_table":
		// if r.RemoteAddr[:9] != "192.168.56.183" {
		// 	http.Error(w, "Unauthorized to modify databases or tables", 403)
		// 	return
		// }
		if cmd.Action == "create_db" || cmd.Action == "drop_db" || cmd.Action == "create_table" || cmd.Action == "drop_table" {
			if r.Header.Get("X-Master-Key") != "secret123" {
				http.Error(w, "Unauthorized to modify databases or tables", 403)
				return
			}
		}

		if cmd.Action == "create_db" {
			_, err := db.Exec("CREATE DATABASE IF NOT EXISTS " + cmd.DBName)
			respond(w, err, "Database created")
			if err == nil {
				replicateToSlaves(cmd) 
			}
		} else if cmd.Action == "drop_db" {
			_, err := db.Exec("DROP DATABASE IF EXISTS " + cmd.DBName)
			respond(w, err, "Database dropped")
			//os.Exit(0)
		} else if cmd.Action == "drop_table" {
			useDB(cmd.DBName)
			_, err := db.Exec("DROP TABLE IF EXISTS " + cmd.Table)
			respond(w, err, "Table dropped")
		} else if cmd.Action == "create_table" {
			_, err := db.Exec("CREATE DATABASE IF NOT EXISTS " + cmd.DBName)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			useDB(cmd.DBName)
			cols := ""
			for _, col := range cmd.Attrs {
				cols += fmt.Sprintf("%s TEXT,", col)
			}
			cols = cols[:len(cols)-1]
			query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)", cmd.Table, cols)
			_, err = db.Exec(query)
			respond(w, err, "Table created")
			if err == nil {
				replicateToSlaves(cmd) 
			}
		}

	case "insert":
		if cmd.DBName == "" || cmd.Table == "" || len(cmd.Data) == 0 {
			http.Error(w, "Missing DBName, Table or Data", 400)
			return
		}
		useDB(cmd.DBName)
		id := cmd.Data["id"]
		var exists int
		db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE id = ?", cmd.Table), id).Scan(&exists)
		if exists > 0 {
			http.Error(w, "ID already exists", 400)
			return
		}
		cols, vals, args := "", "", []interface{}{}
		for k, v := range cmd.Data {
			cols += k + ","
			vals += "?,"
			args = append(args, v)
		}
		cols = cols[:len(cols)-1]
		vals = vals[:len(vals)-1]
		query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", cmd.Table, cols, vals)
		_, err := db.Exec(query, args...)
		respond(w, err, "Inserted")
		if err == nil {
			replicateToSlaves(cmd)
		}

	case "select":
		if cmd.DBName == "" || cmd.Table == "" {
			http.Error(w, "Missing DBName or Table", 400)
			return
		}
		useDB(cmd.DBName)
		query := fmt.Sprintf("SELECT * FROM %s", cmd.Table)
		rows, err := db.Query(query)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		cols, _ := rows.Columns()
		results := []map[string]string{}
		for rows.Next() {
			tmp := make([]interface{}, len(cols))
			tmpPtrs := make([]interface{}, len(cols))
			for i := range tmp {
				tmpPtrs[i] = &tmp[i]
			}
			rows.Scan(tmpPtrs...)
			m := make(map[string]string)
			for i, col := range cols {
				m[col] = fmt.Sprintf("%s", tmp[i])
			}
			results = append(results, m)
		}
		json.NewEncoder(w).Encode(results)

	case "search":
		if cmd.DBName == "" || cmd.Table == "" || cmd.Query == nil {
			http.Error(w, "Missing DBName, Table or Query", 400)
			return
		}
		useDB(cmd.DBName)
		key,ok1 := cmd.Query["key"]
		val,ok2 := cmd.Query["value"]
		if !ok1 || !ok2 {
			http.Error(w, "Query must contain 'key' and 'value'", 400)
			return
		}
		query := fmt.Sprintf("SELECT * FROM %s WHERE %s = ?", cmd.Table, key)
		rows, err := db.Query(query, val)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		cols, _ := rows.Columns()
		results := []map[string]string{}
		for rows.Next() {
			tmp := make([]interface{}, len(cols))
			tmpPtrs := make([]interface{}, len(cols))
			for i := range tmp {
				tmpPtrs[i] = &tmp[i]
			}
			rows.Scan(tmpPtrs...)
			m := make(map[string]string)
			for i, col := range cols {
				m[col] = fmt.Sprintf("%s", tmp[i])
			}
			results = append(results, m)
		}
		json.NewEncoder(w).Encode(results)

	case "update":
		if cmd.DBName == "" || cmd.Table == "" || len(cmd.Data) == 0 || cmd.Query == nil {
			http.Error(w, "Missing DBName, Table, Data or Query", 400)
			return
		}
		useDB(cmd.DBName)
		key,ok1 := cmd.Query["key"]
		val,ok2 := cmd.Query["value"]
		if !ok1 || !ok2 {
			http.Error(w, "Query must contain 'key' and 'value'", 400)
			return
		}
	
		var count int
		db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s = ?", cmd.Table, key), val).Scan(&count)
		if count == 0 {
			http.Error(w, "Record not found for update", http.StatusNotFound)
			return
		}
	
		if _, found := cmd.Data["id"]; found {
			http.Error(w, "Cannot update 'id' field", http.StatusBadRequest)
			return
		}
	
		setStr := ""
		args := []interface{}{}
		for k, v := range cmd.Data {
			setStr += fmt.Sprintf("%s=?,", k)
			args = append(args, v)
		}
	
		if setStr == "" {
			http.Error(w, "No valid fields to update", http.StatusBadRequest)
			return
		}
	
		setStr = setStr[:len(setStr)-1]
		cond := fmt.Sprintf("%s='%s'", key, val)
		query := fmt.Sprintf("UPDATE %s SET %s WHERE %s", cmd.Table, setStr, cond)
		_, err := db.Exec(query, args...)
		respond(w, err, "Updated")
		if err == nil {
			replicateToSlaves(cmd)
		}
	

	case "delete":
		if cmd.DBName == "" || cmd.Table == "" || cmd.Query == nil {
			http.Error(w, "Missing DBName, Table or Query", 400)
			return
		}
		
		useDB(cmd.DBName)
		key := cmd.Query["key"]
		val := cmd.Query["value"]
		var count int
		db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s = ?", cmd.Table, key), val).Scan(&count)
		if count == 0 {
			http.Error(w, "Record not found for delete", 404)
			return
		}
		cond := fmt.Sprintf("%s='%s'", key, val)
		query := fmt.Sprintf("DELETE FROM %s WHERE %s", cmd.Table, cond)
		_, err := db.Exec(query)
		respond(w, err, "Deleted")
		if err == nil {
			replicateToSlaves(cmd)
		}

	default:
		http.Error(w, "Unknown or unauthorized action", 400)
	}
}

func useDB(name string) {
	_, err := db.Exec("USE " + name)
	if err != nil {
		log.Fatalf("Cannot switch to DB %s: %v", name, err)
	}
}

func respond(w http.ResponseWriter, err error, msg string) {
	if err != nil {
		log.Println("ERROR:", err)
		http.Error(w, err.Error(), 500)
		return
	}
	w.Write([]byte(msg))
}

func replicateToSlaves(cmd Command) {
	for _, slaveURL := range slaves {
		go func(url string) {
			jsonData, _ := json.Marshal(cmd)
			resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
			if err != nil {
				log.Println("Replication error to", url, ":", err)
				return
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			log.Printf("Replicated to %s: %s\n", url, string(body))
		}(slaveURL)
	}
}


func main() {
	connectDatabase()
	http.HandleFunc("/master", handleCommand)
	log.Println("Master running on :5000")
	http.ListenAndServe(":5000", nil)
}
