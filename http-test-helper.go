package graphqlkit

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
)

var userID = 1
var secret = []byte("#yuui123")
var expired = false

//CreateTempFile Create a temp file with the data indicated
func CreateTempFile(data string) (*os.File, func(), error) {
	file, err := ioutil.TempFile("./", "temp")
	if err != nil {
		return nil, nil, err
	}
	_, err = file.WriteString(data)
	return file, func() {
		file.Close()
		os.Remove(file.Name())
	}, err
}

//CreateGraphqlRequest Create a Graphql request
func CreateGraphqlRequest(request string) (*http.Request, error) {
	body := fmt.Sprintf("{\"query\":\"{ %s }\"}", request)
	req, err := http.NewRequest("POST", "/graphql", strings.NewReader(body))

	if err != nil {
		log.Printf(err.Error())
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

//CreateGraphqlRequestWithAuthentication Create a Graphql request with authentication token
func CreateGraphqlRequestWithAuthentication(request string) (*http.Request, error) {
	req, err := CreateGraphqlRequest(request)

	if err != nil {
		return nil, err
	}

	tokenString := createJWTToken()

	req.Header.Set("Authorization", "Bearer "+tokenString)
	return req, nil
}

type customClaims struct {
	User string `json:"user"`
	jwt.StandardClaims
}

func createJWTToken() string {
	// Create a new token object, specifying signing method and the claims
	// you would like it to contain.
	var token *jwt.Token
	if expired {
		claims := customClaims{
			"abc",
			jwt.StandardClaims{
				ExpiresAt: time.Now().Add(time.Second * -1).Unix(),
				IssuedAt:  jwt.TimeFunc().Unix(),
			},
		}

		token = jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	} else if userID != 0 {
		token = jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.MapClaims{
			"user": userID,
		})
	} else {
		token = jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.MapClaims{})
	}

	// Sign and get the complete encoded token as a string using the secret
	tokenString, _ := token.SignedString(secret)
	return tokenString
}

// CheckResponseOk Check if the response is 200 Ok
func CheckResponseOk(resp *httptest.ResponseRecorder, t *testing.T) {
	if resp.Code != http.StatusOK {
		t.Errorf("Deveria ter retornado 200 Ok e retornou %v\n", resp.Code)
	}
}

// CheckResponseUnauthorized Check if the response is unauthorized
func CheckResponseUnauthorized(resp *httptest.ResponseRecorder, t *testing.T, err string) {
	if resp.Code != http.StatusUnauthorized {
		t.Errorf("Deveria ter retornado Unauthorized e retornou %v:%s\n", resp.Code, resp.Body.String())
	}

	expected := fmt.Sprintf("{\"err\":\"%s\"}", err)
	if reflect.DeepEqual(resp.Body.String(), expected) {
		t.Errorf("O erro deveria ter sido %s e foi %s", expected, resp.Body.String())
	}
}
