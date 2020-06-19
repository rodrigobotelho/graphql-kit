package graphqlkit

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	gokitjwt "github.com/go-kit/kit/auth/jwt"
	"github.com/go-kit/kit/log"
	graphql "github.com/graph-gophers/graphql-go"
)

type loggingService struct {
	logger log.Logger
	Service
	blacklist map[string]bool
}

// NewLoggingService Create a logging service that logs method, query,
// how much it took and possible erros
func NewLoggingService(logger log.Logger, s Service, blacklist []string) Service {
	bl := make(map[string]bool)
	for _, method := range blacklist {
		bl[strings.ToUpper(method)] = true
	}
	return &loggingService{logger, s, bl}
}

func (s *loggingService) Exec(ctx context.Context, req GraphqlRequest) (res *graphql.Response) {
	defer func(begin time.Time) {
		var responseErr error
		if len(res.Errors) > 0 {
			responseErr = fmt.Errorf("request error: %v", res.Errors)
		}
		if req.OperationName == "" {
			req.OperationName = findOpName(req.Query)
		}
		if responseErr == nil && s.inBlacklist(strings.ToUpper(req.OperationName)) {
			return
		}
		variablesJSON, err := json.Marshal(req.Variables)
		if err != nil {
			variablesJSON = []byte("error marshaling variables to json: " + err.Error())
		}
		responseJSON, err := json.Marshal(res)
		if err != nil {
			responseJSON = []byte("error marshaling response to json: " + err.Error())
		}
		claimsValue := reflect.ValueOf(ctx.Value(gokitjwt.JWTClaimsContextKey))
		fmt.Printf(claimsValue.Kind().String())
		subject := "Not Authenticated"
		if claimsValue.IsValid() {
			subjectValue := claimsValue.Elem().FieldByName("Subject")
			if subjectValue.IsValid() {
				subject = subjectValue.String()
			}
		}
		s.logger.Log(
			"user", subject,
			"method", req.OperationName,
			"query", req.Query,
			"variables", string(variablesJSON),
			"took", time.Since(begin),
			"error", responseErr,
			"response", string(responseJSON),
		)
	}(time.Now())
	res = s.Service.Exec(ctx, req)
	return res
}

func (s *loggingService) inBlacklist(operation string) bool {
	return s.blacklist[operation]
}
