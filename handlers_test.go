package graphqlkit

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/go-kit/kit/log"
	graphql "github.com/graph-gophers/graphql-go"
)

var queryResolver = anyResolver{}
var param = []int{1, 2, 3}
var schema = `schema {
    query: Query
}
type Query {
    anyMethod(param: [ID]!): [ID]
}`
var auth = false
var secretServer string
var logger log.Logger
var hasInstrumenting bool

type anyResolver struct {
	Answer    []int
	ManyCalls int
	Err       error
}

func (qR *anyResolver) AnyMethod(ctx context.Context, args struct{ Param []*graphql.ID }) (*[]*graphql.ID, error) {
	qR.ManyCalls++
	if qR.Err != nil {
		return nil, qR.Err
	}
	retorno := transformIntToGraphQlID(qR.Answer)
	return &retorno, nil
}

func transformIntToGraphQlID(params []int) []*graphql.ID {
	retorno := []*graphql.ID{}
	for _, param := range params {
		graphID := graphql.ID(strconv.Itoa(param))
		retorno = append(retorno, &graphID)
	}
	return retorno
}

func makeAnyService() (req *http.Request, resp *httptest.ResponseRecorder) {
	file, remove, err := CreateTempFile(schema)
	if err != nil {
		return nil, nil
	}
	defer remove()
	var graphqlHander Handlers
	graphqlHander.AddGraphqlService(file.Name(), &queryResolver)
	query := fmt.Sprintf("\"{ anyMethod(param: %v) }\"", param)
	if auth {
		graphqlHander.AddAuthenticationService(secretServer,
			jwt.SigningMethodHS512, &jwt.StandardClaims{})
		req, err = CreateGraphqlRequestWithAuthentication(query)
	} else {
		req, err = CreateGraphqlRequest(query)
	}
	if err != nil {
		return nil, nil
	}

	if logger != nil {
		graphqlHander.AddLoggingService(logger)
	}

	if hasInstrumenting {
		now := fmt.Sprintf("%d", time.Now().UnixNano())
		graphqlHander.AddInstrumentingService("teste", "teste"+now)
	}

	resp = httptest.NewRecorder()
	mux := http.NewServeMux()
	mux.Handle("/graphql", graphqlHander.Handler())

	mux.ServeHTTP(resp, req)
	return req, resp
}

func setup() {
	UserID = 1
	Expired = false
	auth = false
	queryResolver = anyResolver{}
	secretServer = string(Secret)
	logger = nil
	hasInstrumenting = false
}

func TestAnyMethodWithoutAuthentication_callService_ShouldCallResolver(t *testing.T) {
	//Arrange
	setup()
	queryResolver.Answer = make([]int, 0, 1)
	queryResolver.Answer = append(queryResolver.Answer, 1)

	//Act
	_, resp := makeAnyService()

	//Assert
	CheckResponseOk(resp, t)

	if queryResolver.ManyCalls != 1 {
		t.Errorf("The resolver should be call at least once and was %d\n", queryResolver.ManyCalls)
	}
}

func TestAnyMethodWithoutAuthentication_callService_ShouldReturnAnAnswer(t *testing.T) {
	//Arrange
	setup()
	queryResolver.Answer = make([]int, 0, 1)
	queryResolver.Answer = append(queryResolver.Answer, 1)
	queryResolver.Answer = append(queryResolver.Answer, 2)
	queryResolver.Answer = append(queryResolver.Answer, 3)

	//Act
	_, resp := makeAnyService()

	//Assert
	CheckResponseOk(resp, t)

	expected := `{"data":{"anyMethod":["1","2","3"]}}`

	if resp.Body.String() != expected {
		t.Errorf("Should have returned an answer with %s and did %s\n",
			expected, resp.Body.String())
	}
}

func TestAnyMethodWithAuthentication_WithTokenExpired_ShouldReturnUnauthorized(t *testing.T) {
	//Arrange
	setup()
	queryResolver.Answer = make([]int, 0, 1)
	queryResolver.Answer = append(queryResolver.Answer, 1)
	auth = true
	Expired = true

	//Act
	_, resp := makeAnyService()

	//Assert
	CheckResponseUnauthorized(resp, t, "JWT Token is expired")
}

func TestAnyMethodWithAuthentication_WithTokenInvalid_ShouldReturnUnauthorized(t *testing.T) {
	//Arrange
	setup()
	queryResolver.Answer = make([]int, 0, 1)
	queryResolver.Answer = append(queryResolver.Answer, 1)
	auth = true
	secretServer = "somethingelse"

	//Act
	_, resp := makeAnyService()

	//Assert
	CheckResponseUnauthorized(resp, t, "signature is invalid")
}

func TestAnyMethodWithAuthenticationLogging_WithLogger_ShouldLog(t *testing.T) {
	//Arrange
	setup()
	auth = true
	var buf bytes.Buffer
	logger = log.NewLogfmtLogger(&buf)

	//Act
	_, resp := makeAnyService()

	//Assert
	CheckResponseOk(resp, t)

	if len(buf.String()) == 0 {
		t.Error("Should have logged, but it didn't.\n")
	}
}

func TestAnyMethodWithAuthenticationLoggingInstrumetation_CantPanic(t *testing.T) {
	//Arrange
	setup()
	auth = true
	var buf bytes.Buffer
	logger = log.NewLogfmtLogger(&buf)
	hasInstrumenting = true

	//Act
	_, resp := makeAnyService()

	//Assert
	CheckResponseOk(resp, t)
}

func TestAnyMethodWithoutAuthenticationLoggingInstrumetation_CantPanic(t *testing.T) {
	//Arrange
	setup()
	auth = false
	var buf bytes.Buffer
	logger = log.NewLogfmtLogger(&buf)
	hasInstrumenting = true

	//Act
	_, resp := makeAnyService()

	//Assert
	CheckResponseOk(resp, t)
}
