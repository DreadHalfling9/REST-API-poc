package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

var db *pgxpool.Pool

type Todo struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

func initializeDatabase() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL not set")
	}

	var err error
	db, err = pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
}

func getTodos(w http.ResponseWriter, _ *http.Request) {
	rows, err := db.Query(context.Background(), "SELECT id, title, done FROM todos ORDER BY id")
	if err != nil {
		http.Error(w, "Failed to fetch todos", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var todos []Todo
	for rows.Next() {
		var t Todo
		err := rows.Scan(&t.ID, &t.Title, &t.Done)
		if err == nil {
			todos = append(todos, t)
		}
	}

	json.NewEncoder(w).Encode(todos)
}

func getTodoById(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/todos/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var t Todo
	err = db.QueryRow(context.Background(),
		"SELECT id, title, done FROM todos WHERE id = $1", id).
		Scan(&t.ID, &t.Title, &t.Done)

	if err != nil {
		http.NotFound(w, r)
		return
	}

	json.NewEncoder(w).Encode(t)
}

func createTodo(w http.ResponseWriter, r *http.Request) {
	var t Todo
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	err := db.QueryRow(context.Background(),
		"INSERT INTO todos (title, done) VALUES ($1, $2) RETURNING id",
		t.Title, t.Done).Scan(&t.ID)

	if err != nil {
		log.Printf("Failed to insert todo: %v", err)
		http.Error(w, "Failed to insert", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(t)
}

func updateTodo(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/todos/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var updated Todo
	if err := json.NewDecoder(r.Body).Decode(&updated); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	_, err = db.Exec(context.Background(),
		"UPDATE todos SET title = $1, done = $2 WHERE id = $3",
		updated.Title, updated.Done, id)

	if err != nil {
		http.Error(w, "Update failed", http.StatusInternalServerError)
		return
	}

	updated.ID = id
	json.NewEncoder(w).Encode(updated)
}

func markAsDone(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/todos/")
	idStr = strings.TrimSuffix(idStr, "/done")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	_, err = db.Exec(context.Background(), "UPDATE todos SET done = true WHERE id = $1", id)
	if err != nil {
		http.Error(w, "Failed to mark done", http.StatusInternalServerError)
		return
	}

	var t Todo
	err = db.QueryRow(context.Background(), "SELECT id, title, done FROM todos WHERE id = $1", id).
		Scan(&t.ID, &t.Title, &t.Done)

	if err != nil {
		http.NotFound(w, r)
		return
	}

	json.NewEncoder(w).Encode(t)
}

func deleteTodo(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/todos/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	_, err = db.Exec(context.Background(), "DELETE FROM todos WHERE id = $1", id)
	if err != nil {
		http.Error(w, "Delete failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func router(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/todos":
		getTodos(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/todos":
		createTodo(w, r)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/todos/"):
		getTodoById(w, r)
	case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/todos/"):
		updateTodo(w, r)
	case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/todos/"):
		deleteTodo(w, r)
	case r.Method == http.MethodPatch && strings.HasSuffix(r.URL.Path, "/done"):
		markAsDone(w, r)
	default:
		http.NotFound(w, r)
	}
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	initializeDatabase()
	http.HandleFunc("/", router)
	log.Println("Server running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
