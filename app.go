package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	v "github.com/webdeveloppro/validating"
)

// App holding routers and DB connection
type App struct {
	Router  *mux.Router
	Storage Storage
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
func NewApp(storage Storage) (a App, err error) {
	a = App{}
	a.Router = mux.NewRouter()
	a.initializeRoutes()
	a.Storage = storage
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

	if err := a.Storage.GetUserByEmail(&u); err != nil {
		errors["__error__"] = append(errors["__error__"], "email or password do not match")
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

	errs := v.Validate(v.Schema{
		v.F("email", &u.Email):       v.All(v.Nonzero("cannot be empty"), v.Len(4, 120, "length is not between 4 and 120")),
		v.F("password", &u.Password): v.All(v.Nonzero("cannot be empty"), v.Len(4, 120, "length is not between 4 and 120")),
	})

	// We don't want to make database query if we already know email is not valid
	if errs.HasField("email") == false {
		err = a.Storage.GetUserByEmail(&u)
		if err.Error() != sql.ErrNoRows.Error() {
			errs.Extend(v.NewErrors("email", v.ErrUnrecognized, err.Error()))
		}
	}

	if len(errs) > 0 {
		respondWithJSON(w, r, http.StatusBadRequest, errs.JSONErrors())
		return
	}

	if err := a.Storage.CreateUser(&u); err != nil {
		errs.Extend(v.NewErrors("__error__", v.ErrInvalid, "cannot create user, please try again in few minutes"))
		log.Fatalf("insert users errors: %+v", err)
	}

	if len(errs) > 0 {
		respondWithJSON(w, r, http.StatusBadRequest, errs.JSONErrors())
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
