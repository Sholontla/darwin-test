package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
)

type UserResponse struct {
	Name  string `json:"name"`
	UUID  string `json:"uuid"`
	Email string `json:"email"`
}

type User struct {
	Name  UserName  `json:"name"`
	Login UserLogin `json:"login"`
	Email string    `json:"email"`
}

type UserName struct {
	First string `json:"first"`
	Last  string `json:"last"`
	Email string `json:"email"`
}

type UserLogin struct {
	UUID string `json:"uuid"`
}

func main() {
	http.HandleFunc("/", UserHandler)
	log.Fatal(http.ListenAndServe(":3000", nil))
}

func UserHandler(w http.ResponseWriter, r *http.Request) {
	numRequests := 5
	resultsPerRequest := 5000

	allUsers, err := LogicFunc(numRequests, resultsPerRequest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonData, err := json.Marshal(allUsers)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}

func LogicFunc(numRequests, resultsPerRequest int) ([]UserResponse, error) {
	var allUsers []UserResponse

	// Set up Goroutines and channels for concurrent fetching and processing
	var wg sync.WaitGroup
	usersCh := make(chan User, numRequests*resultsPerRequest)
	uniqueUUIDs := make(map[string]bool)
	var mutex sync.Mutex

	// Fetch users concurrently using Goroutines
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			url := fmt.Sprintf("https://randomuser.me/api/?results=%d&gender=female", resultsPerRequest)
			response, err := http.Get(url)
			if err != nil {
				log.Println("Error fetching users:", err)
				return
			}
			defer response.Body.Close()

			body, err := io.ReadAll(response.Body)
			if err != nil {
				log.Println("Error reading response body:", err)
				return
			}

			var userResponse struct {
				Results []User `json:"results"`
			}
			err = json.Unmarshal(body, &userResponse)
			if err != nil {
				log.Println("Error unmarshaling user response:", err)
				return
			}

			for _, user := range userResponse.Results {
				usersCh <- user
			}
		}()
	}

	// Close the users channel once all Goroutines are finished
	go func() {
		wg.Wait()
		close(usersCh)
	}()

	// Process users concurrently using Goroutines
	var processWg sync.WaitGroup
	for i := 0; i < numRequests; i++ {
		processWg.Add(1)
		go func() {
			defer processWg.Done()
			for user := range usersCh {
				mutex.Lock()
				if !uniqueUUIDs[user.Login.UUID] {
					allUsers = append(allUsers, UserResponse{
						Name:  user.Name.First + " " + user.Name.Last,
						UUID:  user.Login.UUID,
						Email: user.Email,
					})
					uniqueUUIDs[user.Login.UUID] = true
				}
				mutex.Unlock()
				// Alternatively, you can log the duplicate UUIDs and skip adding them
				// log.Println("Duplicate UUID:", user.Login.UUID)
			}
		}()
	}

	// Wait for all Goroutines to finish processing
	processWg.Wait()

	return allUsers, nil
}
