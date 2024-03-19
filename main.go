package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

)

type User struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Balance int    `json:"balance"`
}

type Quest struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Cost int    `json:"cost"`
}

type CompletedQuest struct {
	UserID  int `json:"user_id"`
	QuestID int `json:"quest_id"`
}

var db *sql.DB

func createUserHandler(w http.ResponseWriter, r *http.Request) {
	var newUser User
	err := json.NewDecoder(r.Body).Decode(&newUser)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// добавление пользователя в базу данных
	_, err = db.Exec("INSERT INTO users (name, balance) VALUES (?, ?)", newUser.Name, newUser.Balance)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response, _ := json.Marshal(newUser)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(response)
}

func createQuestHandler(w http.ResponseWriter, r *http.Request) {
	var newQuest Quest
	err := json.NewDecoder(r.Body).Decode(&newQuest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// добавление задания в базу данных
	_, err = db.Exec("INSERT INTO quests (name, cost) VALUES (?, ?)", newQuest.Name, newQuest.Cost)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response, _ := json.Marshal(newQuest)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(response)
}

func completeQuestHandler(w http.ResponseWriter, r *http.Request) {
	var completed CompletedQuest
	if err := json.NewDecoder(r.Body).Decode(&completed); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var cost int
	err := db.QueryRow("SELECT cost FROM quests WHERE id = ?", completed.QuestID).Scan(&cost)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	_, err = tx.Exec("INSERT INTO completed_quests (user_id, quest_id) VALUES (?, ?)", completed.UserID, completed.QuestID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec("UPDATE users SET balance = balance - ? WHERE id = ?", cost, completed.UserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func getUserTasksHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "user_id parameter is required", http.StatusBadRequest)
		return
	}

	var user User
	err := db.QueryRow("SELECT id, name, balance FROM users WHERE id = ?", userID).Scan(&user.ID, &user.Name, &user.Balance)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rows, err := db.Query("SELECT q.name, q.cost FROM quests q INNER JOIN completed_quests c ON q.id = c.quest_id WHERE c.user_id = ?", userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var tasks []Quest
	for rows.Next() {
		var task Quest
		if err := rows.Scan(&task.Name, &task.Cost); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		tasks = append(tasks, task)
	}

	response := struct {
		User  User    `json:"user"`
		Tasks []Quest `json:"tasks"`
	}{
		User:  user,
		Tasks: tasks,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func main() {
	var err error
	db, err = sql.Open("mysql", "username:password@tcp(localhost:3306)/dbname")
	if err != nil {
		log.Fatal("Error connecting to the database:", err)
	}
	defer db.Close()

	http.HandleFunc("/users", createUserHandler)
	http.HandleFunc("/quests", createQuestHandler)
	http.HandleFunc("/complete-quest", completeQuestHandler)
	http.HandleFunc("/user/", getUserTasksHandler)

	fmt.Println("Server is running on :8000")
	log.Fatal(http.ListenAndServe(":8000", nil))
}
