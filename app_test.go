package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

type FakeStorage struct {
	data User
	resp string
	code int
}

func (s FakeStorage) GetUserByEmail(u *User) error {
	if u.Email == "exist@user.com" {
		return fmt.Errorf("address already exists, do you want to reset password?")
	}

	if u.Email == "new@user.com" {
		return sql.ErrNoRows
	}

	return fmt.Errorf("email do not match anything, please verify email address")
}

func (s FakeStorage) CreateUser(u *User) error {

	if u.Email == "exist@user.com" {
		return fmt.Errorf("user with such email address already exists")
	}

	if u.Email == "new@user.com" {
		u.ID = 1
		return nil
	}

	return fmt.Errorf("email do not match anything, please verify email address")
}

func SetUp(t *testing.T) *App {
	t.Parallel()
	a, _ := NewApp(&FakeStorage{})
	return &a
}

func executeRequest(a *App, req *http.Request) *httptest.ResponseRecorder {
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	a.Router.ServeHTTP(rr, req)

	return rr
}

func checkResponseCode(t *testing.T, expected int, response *httptest.ResponseRecorder, req *http.Request) {

	if expected != response.Code {
		i, _ := req.GetBody()
		b, err := ioutil.ReadAll(i)
		if err != nil {
			t.Errorf("Expected response code %d. Got %d\n, body read error: %v", expected, response.Code, err)
		}

		var msg interface{}
		err = json.Unmarshal(b, &msg)
		if err != nil {
			t.Errorf("Expected response code %d. Got %d\n, json unmarshal error: %v", expected, response.Code, err)
		}

		t.Errorf("Expected response code %d. Got %d\n, body: %+v", expected, response.Code, msg)
	}
}

func TestLoginOptions(t *testing.T) {
	a := SetUp(t)

	req, _ := http.NewRequest("OPTIONS", "/login", nil)
	response := executeRequest(a, req)
	checkResponseCode(t, 200, response, req)
}

func TestSignUpFail(t *testing.T) {

	a := SetUp(t)
	tests := []FakeStorage{
		FakeStorage{
			data: User{
				Email:    "exist@user.com",
				Password: "",
			},
			resp: `{"email":["address already exists, do you want to reset password?"],"password":["cannot be empty"]}`,
			code: 400,
		},
		FakeStorage{
			data: User{
				Email:    "",
				Password: "",
			},
			resp: `{"email":["cannot be empty"],"password":["cannot be empty"]}`,
			code: 400,
		},
		FakeStorage{
			data: User{
				Email:    "new@user.com",
				Password: "",
			},
			resp: `{"password":["cannot be empty"]}`,
			code: 400,
		},
	}

	for _, test := range tests {
		b := new(bytes.Buffer)
		json.NewEncoder(b).Encode(test.data)

		req, _ := http.NewRequest("POST", "/signup", b)
		response := executeRequest(a, req)

		checkResponseCode(t, test.code, response, req)

		if body := response.Body.String(); body != test.resp {
			t.Errorf("Expected %s but got '%s'", test.resp, body)
		}
	}
}

func TestSignUpOptions(t *testing.T) {
	a := SetUp(t)

	req, _ := http.NewRequest("OPTIONS", "/signup", nil)
	response := executeRequest(a, req)
	checkResponseCode(t, 200, response, req)
}

func TestSignUpSuccess(t *testing.T) {

	a := SetUp(t)
	tests := []FakeStorage{
		FakeStorage{
			data: User{
				Email:    "new@user.com",
				Password: "123123",
			},
			resp: `{"id":1}`,
			code: 201,
		},
	}

	for _, test := range tests {
		b := new(bytes.Buffer)
		json.NewEncoder(b).Encode(test.data)

		req, _ := http.NewRequest("POST", "/signup", b)
		response := executeRequest(a, req)

		checkResponseCode(t, test.code, response, req)

		if body := response.Body.String(); body != test.resp {
			t.Errorf("Expected %s but got '%s'", test.resp, body)
		}
	}
}
