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
const bookServiceURL = "http://book-service.default.svc.cluster.local:5001"
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
        http.Error(w, "Call to Book service failed", http.StatusBadRequest)
        return
    }
    defer resp.Body.Close()
    var bookData struct {
        Available bool `json:"available"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&bookData); err != nil {
        http.Error(w, "Error parsing book details received", http.StatusInternalServerError)
        return
    }
    if !bookData.Available {
        http.Error(w, "Book is already borrowed, kindly check after a few days", http.StatusBadRequest)
        return
    }
    stmt, err := db.DB.Prepare("INSERT INTO borrowings (user_id, book_id, borrow_date) VALUES (?, ?, NOW())")
    if err != nil {
        http.Error(w, fmt.Sprintf("DB query preparation failed: %v", err), http.StatusInternalServerError)
        return
    }
    defer stmt.Close()
    _, err = stmt.Exec(request.UserID, request.BookID)
    if err != nil {
        http.Error(w, "Failed to borrow book", http.StatusInternalServerError)
        return
    }
    // Call book service to update availability
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
    row := db.DB.QueryRow("SELECT COUNT(*) FROM borrowings WHERE user_id=? AND book_id=?", request.UserID, request.BookID)
    var count int
    if err := row.Scan(&count); err != nil || count == 0 {
        http.Error(w, "Cannot return a book that was never borrowed", http.StatusBadRequest)
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
    // Call book service to update availability
    updateBookURL := fmt.Sprintf("%s/books/%s", bookServiceURL, request.BookID)
    jsonBody := bytes.NewBuffer([]byte(`{"available":true}`))
	req, _ := http.NewRequest("PUT", updateBookURL, jsonBody)
    httpClient.Do(req)
    // Publish event to Kafka
    message := fmt.Sprintf(`{"bookId":"%s","userId":"%s","message":"Book is available."}`, request.BookID, request.UserID)
    db.PublishEvent("book.available_reserved", message)

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
        message, _ := json.Marshal(map[string]interface{}{
            "userId": userID,
            "bookId": info.BookID,
            "dueDate": info.DueDate,
            "notification": "Your book is overdue!",
        })
        db.PublishEvent("book.overdue", string(message))
    }
    if len(overdueBorrowings) == 0 {
        http.Error(w, "No overdue borrowings found", http.StatusNotFound)
        return
    }
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(overdueBorrowings)
}


