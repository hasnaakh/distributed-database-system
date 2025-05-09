# 🗃️ Distributed Database System

## 🧩 Project Overview

A simple distributed database management system using **Go (master server)** and **HTML/JS frontend**. Supports database/table/data operations with basic access control and CORS handling

A lightweight, distributed MySQL management system with:
- 🔁 Master-Slave replication via HTTP
- 🖥️ Web UI for DB, table, and data operations
- 📡 Automatic slave registration
- 🧩 Modular API-driven structure

---

## 📐 Architecture

```
      +-------------+           POST /slave            +-------------+
      |             | <------------------------------> |             |
      |   Master    |                                 |    Slave     |
      |   Server    | -----> Replication via HTTP ----|   Server(s)  |
      | (Go + MySQL)|                                 | (Go + MySQL) |
      +-------------+                                 +-------------+
            ↑
            |
     POST /master
            ↑
         Web UI (index.html)
```

- **Master** handles all client requests and replicates changes to slaves.
- **Slaves** execute commands such as `SELECT`, `INSERT`, `UPDATE`, `DELETE`, `SEARCH`.
- **Web UI** sends actions to the master server via RESTful requests.

---

## 🚀 Setup Instructions

### Prerequisites
- Go 1.19+
- MySQL Server
- Modern web browser (for `index.html`)

### 1. Clone the Repository
```bash
git clone https://github.com/hasnaakh/distributed-database-system.git
cd distributed-database-system
```

### 2. Start MySQL Server
Ensure MySQL is running and credentials are correct in:
- `master.go`: `root:password of master device`
- `slave.go`: `root:password of slave device`

Update if necessary in `sql.Open(...)`.


### 3. Define Slave IPs in `master.go`

Open the `master.go` file and specify the IPs of all slave devices that will receive replicated commands:

```go
var slaves = []string{
    "http://IP of slave 1:5001/slave", //Slave 1
	"http://IP of slave 2:5001/slave", //Slave 2
}
```

> Make sure these addresses are reachable from the master machine.


### 4. Define Master IP in `slave.go`

In the `slave.go` file, update the line where the slave "pings" the master. This is how it registers itself:

```go
resp, err := http.Post("http://IP of master device:5000/master", "application/json", bytes.NewBuffer(jsonData))
```

> Replace `IP of master device` with the actual IP address of the master node in your network such as `192.168.1.9`.

### 5. Run Master
```bash
go run master.go
```
Master listens on `http://localhost:5000/master`

### 6. Launch UI

To launch the frontend interface:

#### Option 1 (Recommended): Use `http-server`
```bash
npm install -g http-server
http-server . -p 5050
```
Then open: [http://localhost:5050/index.html](http://localhost:5050/index.html)

#### Option 2: Use Python HTTP server
```bash
python -m http.server 5050
```
Then open: [http://localhost:5050/index.html](http://localhost:5050/index.html)



### 7. Run Slave(s)
```bash
cd slave
```
```bash
go run slave.go
```
> You can run this on different IPs or machines to simulate multiple slaves.


---

## 📂 File Structure

```
.
├── master.go        # Master server with replication logic
├── slave
│   └── slave.go         # Slave node to execute DB commands
├── index.html       # Web interface to control system
```

---

## 🧪 Example Usage

### Create a Database
```json
{
  "action": "create_db",
  "dbname": "school"
}
```
### Drop Database

```json
{
  "action": "drop_db",
  "dbname": "school"
}
```

### Create a Table
```json
{
  "action": "create_table",
  "dbname": "school",
  "table": "students",
  "attrs": ["id", "name", "grade"]
}
```

### Insert Data
```json
{
  "action": "insert",
  "dbname": "school",
  "table": "students",
  "data": {
    "id": "1",
    "name": "Ali",
    "grade": "A"
  }
}
```

### Update Data
```json
{
  "action": "update",
  "dbname": "school",
  "table": "students",
  "query": { "key": "id", "value": "1" },
  "data": { "grade": "B" }
}
```
### Drop Table

```json
{
  "action": "drop_table",
  "dbname": "school",
  "table": "students"
}
```

### Select Data

```json
{
  "action": "select",
  "dbname": "school",
  "table": "students"
}
```

### Search

```json
{
  "action": "search",
  "dbname": "school",
  "table": "students",
  "query": { "key": "grade", "value": "A" }
}
```

### Delete

```json
{
  "action": "delete",
  "dbname": "school",
  "table": "students",
  "query": { "key": "id", "value": "1" }
}
```



---

## ⚙️ Core Features

- 🔐 Basic auth for destructive commands via `X-Master-Key`
- 🌐 CORS enabled for browser access
- 📦 Modular design (easy to extend)
- 🔄 Full replication of insert/update/create operations

---

## 📣 Notes

- Only `master` accepts UI/HTTP requests. Slaves respond only to master's replication calls.
- Each slave automatically "pings" the master upon start.

---

## 🏗️ Architecture & Design

This system follows a simple **Master-Slave** architecture. The master node acts as the controller and synchronizes commands with registered slaves.

### 2. Communication Protocol

All nodes communicate using JSON-formatted POST, GET, Delete requests over HTTP. Slaves automatically send a "ping" command to the master to register themselves.

Each request contains:
- `action`: The operation type (e.g., create_db, insert)
- `dbname`: Target database (if applicable)
- `table`: Target table (if applicable)
- `data`: Key-value pairs (used in insert/update)
- `query`: Conditions (used in update/delete/search)
- `attrs`: List of attributes (used in create_table)

---


## 🛠️ Features & Functionalities

| Operation       | Method  | Accessible by | Description                                 |
|----------------|---------|---------------|---------------------------------------------|
| `create_db`     | POST    | Master only   | Creates a new database                      |
| `drop_db`       | DELETE  | Master only   | Deletes an existing database                |
| `create_table`  | POST    | Master only   | Creates a table with user-defined columns   |
| `drop_table`    | DELETE  | Master only   | Deletes a table                             |
| `insert`        | POST    | Any node      | Adds a record to a table                    |
| `select`        | GET     | Any node      | Retrieves all data from a table             |
| `search`        | GET     | Any node      | Finds specific records by key/value         |
| `update`        | PUT     | Any node      | Updates records in a table                  |
| `delete`        | DELETE  | Any node      | Deletes records from a table                |



---




