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
	mutation: Mutation
}
type Query {
    anyMethod(param: [ID]!): [ID]
}
type Mutation {
	anyMethod2(param: [ID]!): Boolean
}`

type testOptions struct {
	auth             bool
	secretServer     string
	logger           log.Logger
	hasInstrumenting bool
	addBlacklist     []string
	mutation         bool
}

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

func (qR *anyResolver) AnyMethod2(ctx context.Context, args struct{ Param []*graphql.ID }) (*bool, error) {
	qR.ManyCalls++
	ret := true
	return &ret, qR.Err
}

func transformIntToGraphQlID(params []int) []*graphql.ID {
	retorno := []*graphql.ID{}
	for _, param := range params {
		graphID := graphql.ID(strconv.Itoa(param))
		retorno = append(retorno, &graphID)
	}
	return retorno
}

func (tst *testOptions) makeAnyService() (req *http.Request, resp *httptest.ResponseRecorder) {
	file, remove, err := CreateTempFile(schema)
	if err != nil {
		return nil, nil
	}
	defer remove()
	var graphqlHander Handlers
	graphqlHander.AddGraphqlService(file.Name(), &queryResolver)
	if len(tst.addBlacklist) != 0 {
		graphqlHander.AddLoggingBlacklist(tst.addBlacklist)
	}
	var query string
	if tst.mutation {
		query = fmt.Sprintf("{ mutation: anyMethod2(param: %v) }", param)
	} else {
		query = fmt.Sprintf("{ anyMethod(param: %v) }", param)
	}
	if tst.auth {
		graphqlHander.AddAuthenticationService(tst.secretServer,
			jwt.SigningMethodHS512, &jwt.StandardClaims{})
		req, err = CreateGraphqlRequestWithAuthentication(query)
	} else {
		req, err = CreateGraphqlRequest(query)
	}
	if err != nil {
		return nil, nil
	}

	if tst.logger != nil {
		graphqlHander.AddLoggingService(tst.logger)
	}

	if tst.hasInstrumenting {
		now := fmt.Sprintf("%d", time.Now().UnixNano())
		graphqlHander.AddInstrumentingService("teste", "teste"+now)
	}

	resp = httptest.NewRecorder()
	mux := http.NewServeMux()
	mux.Handle("/graphql", graphqlHander.Handler())

	mux.ServeHTTP(resp, req)
	return req, resp
}

func setup() *testOptions {
	UserID = 1
	Expired = false
	queryResolver = anyResolver{}
	return &testOptions{
		auth:             false,
		secretServer:     string(Secret),
		logger:           nil,
		hasInstrumenting: false,
		mutation:         false,
	}
}

func TestAnyMethodWithoutAuthentication_callService_ShouldCallResolver(t *testing.T) {
	//Arrange
	tst := setup()
	queryResolver.Answer = make([]int, 0, 1)
	queryResolver.Answer = append(queryResolver.Answer, 1)

	//Act
	_, resp := tst.makeAnyService()

	//Assert
	CheckResponseOk(resp, t)

	if queryResolver.ManyCalls != 1 {
		t.Errorf("The resolver should be call at least once and was %d\n", queryResolver.ManyCalls)
	}
}

func TestAnyMethodWithoutAuthentication_callService_ShouldReturnAnAnswer(t *testing.T) {
	//Arrange
	tst := setup()
	queryResolver.Answer = make([]int, 0, 1)
	queryResolver.Answer = append(queryResolver.Answer, 1)
	queryResolver.Answer = append(queryResolver.Answer, 2)
	queryResolver.Answer = append(queryResolver.Answer, 3)

	//Act
	_, resp := tst.makeAnyService()

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
	tst := setup()
	queryResolver.Answer = make([]int, 0, 1)
	queryResolver.Answer = append(queryResolver.Answer, 1)
	tst.auth = true
	Expired = true

	//Act
	_, resp := tst.makeAnyService()

	//Assert
	CheckResponseUnauthorized(resp, t, "JWT Token is expired")
}

func TestAnyMethodWithAuthentication_WithTokenInvalid_ShouldReturnUnauthorized(t *testing.T) {
	//Arrange
	tst := setup()
	queryResolver.Answer = make([]int, 0, 1)
	queryResolver.Answer = append(queryResolver.Answer, 1)
	tst.auth = true
	tst.secretServer = "somethingelse"

	//Act
	_, resp := tst.makeAnyService()

	//Assert
	CheckResponseUnauthorized(resp, t, "signature is invalid")
}

func TestAnyMethodWithAuthenticationLogging_WithLogger_ShouldLog(t *testing.T) {
	//Arrange
	tst := setup()
	tst.auth = true
	var buf bytes.Buffer
	tst.logger = log.NewLogfmtLogger(&buf)

	//Act
	_, resp := tst.makeAnyService()

	//Assert
	CheckResponseOk(resp, t)

	if len(buf.String()) == 0 {
		t.Error("Should have logged, but it didn't.\n")
	}
}

func TestAnyMethodInBlacklist_WithLogger_ShouldNotLog(t *testing.T) {
	//Arrange
	tst := setup()
	tst.auth = true
	var buf bytes.Buffer
	tst.logger = log.NewLogfmtLogger(&buf)
	tst.addBlacklist = append(tst.addBlacklist, "anyMethod")
	//Act
	_, resp := tst.makeAnyService()

	//Assert
	CheckResponseOk(resp, t)

	if len(buf.String()) != 0 {
		t.Error("Shouldn't have logged, but it did.\n")
	}
}

func TestMutationAnyMethod2InBlacklist_WithLogger_ShouldNotLog(t *testing.T) {
	//Arrange
	tst := setup()
	tst.auth = true
	var buf bytes.Buffer
	tst.logger = log.NewLogfmtLogger(&buf)
	tst.addBlacklist = append(tst.addBlacklist, "anyMethod2")
	tst.mutation = true

	//Act
	_, resp := tst.makeAnyService()

	//Assert
	CheckResponseOk(resp, t)

	if len(buf.String()) != 0 {
		t.Error("Shouldn't have logged, but it did.\n")
	}
}

func TestAnyMethodWithAuthenticationLoggingInstrumetation_CantPanic(t *testing.T) {
	//Arrange
	tst := setup()
	tst.auth = true
	var buf bytes.Buffer
	tst.logger = log.NewLogfmtLogger(&buf)
	tst.hasInstrumenting = true

	//Act
	_, resp := tst.makeAnyService()

	//Assert
	CheckResponseOk(resp, t)
}

func TestAnyMethodWithoutAuthenticationLoggingInstrumetation_CantPanic(t *testing.T) {
	//Arrange
	tst := setup()
	tst.auth = false
	var buf bytes.Buffer
	tst.logger = log.NewLogfmtLogger(&buf)
	tst.hasInstrumenting = true

	//Act
	_, resp := tst.makeAnyService()

	//Assert
	CheckResponseOk(resp, t)
}
