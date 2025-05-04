package controllers

import (
    "borrow-service/internal/db"
    "encoding/json"
    "net/http"
    "fmt"
    "time"
    "github.com/gorilla/mux"
    "bytes"
)

type BorrowingInfo struct {
    BookID    string    `json:"bookId"`
    DueDate   time.Time `json:"dueDate"`
}
const bookServiceURL = "http://book-service.default.svc.cluster.local:5000"
var httpClient = &http.Client{}

func BorrowBook(w http.ResponseWriter, r *http.Request) {
    var request struct {
        UserID string `json:"userId"`
        BookID string `json:"bookId"`
    }
    if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
        http.Error(w, "Invalid request format", http.StatusBadRequest)
        return
    }
    bookURL := fmt.Sprintf("%s/books/%s", bookServiceURL, request.BookID)
    resp, err := httpClient.Get(bookURL)
    if err != nil || resp.StatusCode != http.StatusOK {
        http.Error(w, "Book not available", http.StatusBadRequest)
        return
    }
    defer resp.Body.Close()
    stmt, err := db.DB.Prepare("INSERT INTO borrowings (user_id, book_id, borrow_date) VALUES (?, ?, NOW())")
    if err != nil {
        http.Error(w, "DB query preparation failed", http.StatusInternalServerError)
        return
    }
    defer stmt.Close()
    _, err = stmt.Exec(request.UserID, request.BookID)
    if err != nil {
        http.Error(w, "Failed to borrow book", http.StatusInternalServerError)
        return
    }
    jsonBody := bytes.NewBuffer([]byte(`{"available":false}`))
    updateBookURL := fmt.Sprintf("%s/books/%s", bookServiceURL, request.BookID)
	req, _ := http.NewRequest("PUT", updateBookURL, jsonBody)
    httpClient.Do(req)
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{"message": "Book borrowed successfully"})
}

func ReturnBook(w http.ResponseWriter, r *http.Request) {
    var request struct {
        UserID string `json:"userId"`
        BookID string `json:"bookId"`
    }
    if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
        http.Error(w, "Invalid request format", http.StatusBadRequest)
        return
    }
    stmt, err := db.DB.Prepare("DELETE FROM borrowings WHERE user_id=? AND book_id=?")
    if err != nil {
        http.Error(w, "DB query preparation failed", http.StatusInternalServerError)
        return
    }
    defer stmt.Close()
	_, err = stmt.Exec(request.UserID, request.BookID)
    if err != nil {
        http.Error(w, "Failed to return book", http.StatusInternalServerError)
        return
    }
    updateBookURL := fmt.Sprintf("%s/books/%s", bookServiceURL, request.BookID)
    jsonBody := bytes.NewBuffer([]byte(`{"available":true}`))
	req, _ := http.NewRequest("PUT", updateBookURL, jsonBody)
    httpClient.Do(req)
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{"message": "Book returned successfully"})
}

func GetBorrowingsByUser(w http.ResponseWriter, r *http.Request) {
    userID := mux.Vars(r)["userId"]
    rows, err := db.DB.Query("SELECT book_id, borrow_date FROM borrowings WHERE user_id=?", userID)
    if err != nil {
        http.Error(w, "Failed to fetch borrowings", http.StatusInternalServerError)
        return
    }
    defer rows.Close()
    var borrowings []BorrowingInfo
    for rows.Next() {
        var info BorrowingInfo
        var borrowDate time.Time
        if err := rows.Scan(&info.BookID, &borrowDate); err != nil {
            http.Error(w, "Data error", http.StatusInternalServerError)
            return
        }
        info.DueDate = borrowDate.Add(14 * 24 * time.Hour)
        borrowings = append(borrowings, info)
    }
    if len(borrowings) == 0 {
        http.Error(w, "No borrowings found for the user", http.StatusNotFound)
        return
    }
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(borrowings)
}

func GetOverdueBorrowings(w http.ResponseWriter, r *http.Request) {
    rows, err := db.DB.Query("SELECT user_id, book_id, borrow_date FROM borrowings WHERE borrow_date < DATE_SUB(NOW(), INTERVAL 14 DAY);")
    if err != nil {
        http.Error(w, "Failed to fetch borrowings", http.StatusInternalServerError)
        return
    }
    defer rows.Close()
    overdueBorrowings := make(map[string][]BorrowingInfo)
    for rows.Next() {
        var userID string
        var info BorrowingInfo
        var borrowDate time.Time
        if err := rows.Scan(&userID, &info.BookID, &borrowDate); err != nil {
            http.Error(w, "Data error", http.StatusInternalServerError)
            return
        }
        info.DueDate = borrowDate.Add(14 * 24 * time.Hour)
        overdueBorrowings[userID] = append(overdueBorrowings[userID], info)
    }
    if len(overdueBorrowings) == 0 {
        http.Error(w, "No overdue borrowings found", http.StatusNotFound)
        return
    }
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(overdueBorrowings)
}


