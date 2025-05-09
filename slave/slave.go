package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"database/sql"

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

var db *sql.DB

func connectDatabase() {
	var err error
	db, err = sql.Open("mysql", "root:password of slave device@tcp(localhost:3306)/") // Replace with actual MySQL credentials
	if err != nil {
		log.Fatal("Failed to connect to DB:", err)
	}
}

func useDB(name string) {
	_, err := db.Exec("USE " + name)
	if err != nil {
		log.Fatalf("Cannot switch to DB %s: %v", name, err)
	}
}

func handleCommand(w http.ResponseWriter, r *http.Request) {
	var cmd Command
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		http.Error(w, "Invalid command", http.StatusBadRequest)
		return
	}
	log.Printf("Received command: %+v", cmd)

	if cmd.Action != "create_db" && cmd.DBName != "" {
		useDB(cmd.DBName)
	}

	switch cmd.Action {
	case "create_db":
		if cmd.DBName == "" {
			http.Error(w, "DBName is required", 400)
			return
		}
		_, err := db.Exec("CREATE DATABASE IF NOT EXISTS " + cmd.DBName)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Write([]byte("Database created"))

	case "create_table":
		if cmd.DBName == "" || cmd.Table == "" || len(cmd.Attrs) == 0 {
			http.Error(w, "Missing DBName, Table or Attributes", 400)
			return
		}
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
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Write([]byte("Table created"))

	case "insert":
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
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Write([]byte("Inserted"))

	case "update":
		key := cmd.Query["key"]
		val := cmd.Query["value"]
		setStr := ""
		args := []interface{}{}
		for k, v := range cmd.Data {
			setStr += fmt.Sprintf("%s=?,", k)
			args = append(args, v)
		}
		if setStr == "" {
			http.Error(w, "No valid fields to update", 400)
			return
		}
		setStr = setStr[:len(setStr)-1]
		query := fmt.Sprintf("UPDATE %s SET %s WHERE %s='%s'", cmd.Table, setStr, key, val)
		_, err := db.Exec(query, args...)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Write([]byte("Updated"))

	case "delete":
		key := cmd.Query["key"]
		val := cmd.Query["value"]
		query := fmt.Sprintf("DELETE FROM %s WHERE %s='%s'", cmd.Table, key, val)
		_, err := db.Exec(query)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Write([]byte("Deleted"))

	default:
		http.Error(w, "Unsupported action", 400)
	}

}

func sendConnectionSignal() {
	cmd := Command{
		Action: "ping",
		DBName: "",
	}
	jsonData, _ := json.Marshal(cmd)
	resp, err := http.Post("http://IP of master device:5000/master", "application/json", bytes.NewBuffer(jsonData)) // Replace with actual IP Address of master device
	if err != nil {
		fmt.Println("Error sending connection signal:", err)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	fmt.Println("Connection Response:", string(body))
}

func main() {
	connectDatabase()
	http.HandleFunc("/slave", handleCommand)
	go sendConnectionSignal()
	fmt.Println("Slave started and ready")
	http.ListenAndServe(":5001", nil)
}
