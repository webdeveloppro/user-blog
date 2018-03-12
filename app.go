package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// App holding routers and DB connection
type App struct {
	Router *mux.Router
	DB     *sql.DB
}

// User information
type User struct {
	ID        int32     `json:"id"`
	Email     string    `json:"email"`
	Password  string    `json:"password,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	LastLogin time.Time `json:"last_login,omitempty"`
}

// NewApp will create new App instance and setup storage connection
func New(host, user, password, dbname string) (a App, err error) {
	a = App{}

	if host == "" {
		log.Fatal("Empty host string, setup DB_HOST env")
		host = "localhost"
	}

	if user == "" {
		return a, fmt.Errorf("Empty user string, setup DB_USER env")
	}

	if dbname == "" {
		return a, fmt.Errorf("Empty dbname string, setup DB_DBNAME env")
	}

	connectionString :=
		fmt.Sprintf("host=%s user=%s password='%s' dbname=%s sslmode=disable", host, user, password, dbname)

	a.DB, err = sql.Open("postgres", connectionString)
	if err != nil {
		return a, fmt.Errorf("Cannot open postgresql connection: %v", err)
	}

	a.Router = mux.NewRouter()
	a.initializeRoutes()
	return a, nil
}

// Run application on 8080 port
func (a *App) Run(addr string) {

	if addr == "" {
		addr = "8000"
	}

	log.Fatal(http.ListenAndServe(":"+addr, a.Router))
}

// initializeRoutes - creates routers, runs automatically in Initialize
func (a *App) initializeRoutes() {
	a.Router.HandleFunc("/login", a.login).Methods("POST")
	a.Router.HandleFunc("/login", a.loginOptions).Methods("OPTIONS")
	a.Router.HandleFunc("/signup", a.signup).Methods("POST")
	a.Router.HandleFunc("/signup", a.signupOptions).Methods("OPTIONS")
}

func (a *App) login(w http.ResponseWriter, r *http.Request) {
	var u User
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&u)

	if err != nil {
		log.Fatalf("cannot decode signup body: %v", err)
	}

	errors := make(map[string][]string, 0)
	if u.Email == "" {
		errors["email"] = append(errors["email"], "email cannot be empty")
	}
	if u.Password == "" {
		errors["password"] = append(errors["password"], "password cannot be empty")
	}

	if len(errors) > 0 {
		respondWithJSON(w, r, http.StatusBadRequest, errors)
		return
	}

	if err := a.DB.QueryRow("SELECT id, password FROM users WHERE email=$1", u.Email).Scan(&u.ID, &u.Password); err != nil {
		errors["__error__"] = append(errors["__error__"], "email not found")
	}

	if len(errors) > 0 {
		respondWithJSON(w, r, http.StatusBadRequest, errors)
		return
	}

	respondWithJSON(w, r, 200, u)
}

func (a *App) loginOptions(w http.ResponseWriter, r *http.Request) {
	respondWithJSON(w, r, 200, map[string]map[string]string{
		"email":    map[string]string{"type": "string", "required": "1", "maxLength": "255"},
		"password": map[string]string{"type": "password", "required": "1", "maxLength": "255"},
	})
}

func (a *App) signup(w http.ResponseWriter, r *http.Request) {
	u := User{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&u)

	if err != nil {
		log.Fatalf("cannot decode signup body: %v", err)
	}

	errors := make(map[string]string)
	if u.Email == "" {
		errors["email"] = "Cannot be empty"
	}

	if u.Password == "" {
		errors["password"] = "Cannot be empty"
	}

	// We don't want to make database query if we already know email is not valid
	if _, ok := errors["email"]; ok == false {
		if err = a.DB.QueryRow("SELECT id, password FROM users WHERE email=$1",
			u.Email,
		).Scan(&u.ID, &u.Password); err != sql.ErrNoRows {
			errors["email"] = err.Error()
		}
	}

	if len(errors) > 0 {
		respondWithJSON(w, r, http.StatusBadRequest, errors)
		return
	}

	if err := a.DB.QueryRow("INSERT INTO users(email, password) VALUES($1, $2) RETURNING id",
		u.Email,
		u.Password,
	).Scan(&u.ID); err != nil {
		errors["__error__"] = "cannot create user, please try again in few minutes"
		log.Fatalf("insert users errors: %+v", err)
	}

	if len(errors) > 0 {
		respondWithJSON(w, r, http.StatusBadRequest, errors)
	} else {
		respondWithJSON(w, r, http.StatusCreated, map[string]int32{"id": u.ID})
	}
}

func (a *App) signupOptions(w http.ResponseWriter, r *http.Request) {
	respondWithJSON(w, r, 200, map[string]map[string]string{
		"email":     map[string]string{"type": "string", "required": "1", "maxLength": "255"},
		"password":  map[string]string{"type": "password", "required": "1", "maxLength": "255"},
		"password2": map[string]string{"type": "password", "required": "1", "maxLength": "255"},
	})
}

func respondWithError(w http.ResponseWriter, r *http.Request, code int, message string) {
	respondWithJSON(w, r, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, r *http.Request, code int, payload interface{}) {
	response, err := json.Marshal(payload)

	if err != nil {
		log.Fatalf("Cannot convert data to json, %v", err)
	}

	if origin := r.Header.Get("Origin"); origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers",
			"Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-REAL")
		w.Header().Set("Content-Type", "application/json")
	}

	// Stop here if its Preflighted OPTIONS request
	if r.Method == "OPTIONS" && r.Header.Get("Accept") == "*/*" {
		return
	}

	respondWithBytes(w, code, response)
}

func respondWithBytes(w http.ResponseWriter, code int, response []byte) {
	w.WriteHeader(code)
	w.Write(response)
}
