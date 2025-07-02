package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type Todo struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

var (
	todos  = []Todo{}
	nextID = 1
	dbFile = "todos.json"
)

func saveTodosToFile() {
	data, err := json.MarshalIndent(todos, "", "  ")
	if err != nil {
		log.Printf("Error marshaling todos: %v", err)
		return
	}
	if err := os.WriteFile(dbFile, data, 0644); err != nil {
		log.Printf("Error writing todos to file: %v", err)
	}
}

func loadTodosFromFile() {
	file, err := os.ReadFile(dbFile)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		log.Fatalf("Error reading file: %v", err)
	}
	if err := json.Unmarshal(file, &todos); err != nil {
		log.Fatalf("Error parsing file: %v", err)
	}
	for _, t := range todos {
		if t.ID >= nextID {
			nextID = t.ID + 1
		}
	}
}

func getTodos(w http.ResponseWriter, _ *http.Request) {
	json.NewEncoder(w).Encode(todos)
}

func getTodo(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/todos/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	for _, t := range todos {
		if t.ID == id {
			json.NewEncoder(w).Encode(t)
			return
		}
	}
	http.NotFound(w, r)
}

func createTodo(w http.ResponseWriter, r *http.Request) {
	var t Todo
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	t.ID = nextID
	nextID++
	todos = append(todos, t)
	saveTodosToFile()
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
	for i, t := range todos {
		if t.ID == id {
			updated.ID = id
			todos[i] = updated
			saveTodosToFile()
			json.NewEncoder(w).Encode(updated)
			return
		}
	}
	http.NotFound(w, r)
}

func markAsDone(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/todos/")
	idStr = strings.TrimSuffix(idStr, "/done")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	for i, t := range todos {
		if t.ID == id {
			todos[i].Done = true
			saveTodosToFile()
			json.NewEncoder(w).Encode(todos[i])
			return
		}
	}
	http.NotFound(w, r)
}

func deleteTodo(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/todos/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	for i, t := range todos {
		if t.ID == id {
			todos = append(todos[:i], todos[i+1:]...)
			saveTodosToFile()
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}
	http.NotFound(w, r)
}

func router(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/todos":
		getTodos(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/todos":
		createTodo(w, r)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/todos/"):
		getTodo(w, r)
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
	loadTodosFromFile()
	http.HandleFunc("/", router)
	log.Println("Server listening at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
