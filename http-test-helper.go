package graphqlkit

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	jwt "github.com/golang-jwt/jwt/v4"
)

var UserID = 1
var Audience = ""
var Secret = []byte("#yuui123")
var Expired = false

var UsingCustom bool

// CreateTempFile Create a temp file with the data indicated
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

// CreateGraphqlRequest Create a Graphql request
func CreateGraphqlRequest(request string) (*http.Request, error) {
	var body string
	if request[0] == '"' {
		body = fmt.Sprintf("{\"query\":%s}", request)
	} else {
		body = fmt.Sprintf("{\"query\":\"%s\"}", request)
	}
	req, err := http.NewRequest("POST", "/graphql", strings.NewReader(body))

	if err != nil {
		log.Printf(err.Error())
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

// CreateGraphqlRequestWithAuthentication Create a Graphql request with authentication token
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
	if Expired {
		claims := customClaims{
			"abc",
			jwt.StandardClaims{
				ExpiresAt: time.Now().Add(time.Second * -1).Unix(),
				IssuedAt:  jwt.TimeFunc().Unix(),
			},
		}

		token = jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	} else if UserID != 0 {
		var claim jwt.Claims
		standard := jwt.StandardClaims{
			Subject: strconv.Itoa(UserID),
		}
		if Audience != "" {
			standard.Audience = Audience
		}
		if UsingCustom {
			claim = customClaims{
				"abc",
				standard,
			}
		} else {
			claim = standard
		}
		token = jwt.NewWithClaims(jwt.SigningMethodHS512, claim)
	} else {
		token = jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.MapClaims{})
	}

	// Sign and get the complete encoded token as a string using the secret
	tokenString, _ := token.SignedString(Secret)
	return tokenString
}

// CheckResponseOk Check if the response is 200 Ok
func CheckResponseOk(resp *httptest.ResponseRecorder, t *testing.T) {
	if resp.Code != http.StatusOK {
		t.Errorf(
			"Deveria ter retornado 200 Ok e retornou %v - %s\n",
			resp.Code,
			resp.Body.String(),
		)
	}
}

// CheckResponseUnauthorized Check if the response is unauthorized
func CheckResponseUnauthorized(resp *httptest.ResponseRecorder, t *testing.T, err string) {
	if resp.Code != http.StatusUnauthorized {
		t.Errorf("Deveria ter retornado Unauthorized e retornou %v:%s\n", resp.Code, resp.Body.String())
	}

	expected := fmt.Sprintf("{\"err\":\"%s\"}\n", err)
	if strings.Compare(resp.Body.String(), expected) != 0 {
		t.Errorf("O erro deveria ter sido %s e foi %s", expected, resp.Body.String())
	}
}
