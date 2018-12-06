package graphqlkit

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

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
		bl[method] = true
	}
	return &loggingService{logger, s, bl}
}

func (s *loggingService) Exec(ctx context.Context, req GraphqlRequest) (res *graphql.Response) {
	defer func(begin time.Time) {
		var err error
		if len(res.Errors) > 0 {
			err = fmt.Errorf("request error: %v", res.Errors)
		}
		if req.OperationName == "" {
			req.OperationName = findOpName(req.Query)
		}
		if s.inBlacklist(req.OperationName) {
			return
		}
		responseJSON, err := json.Marshal(res)
		s.logger.Log(
			"method", req.OperationName,
			"query", req.Query,
			"took", time.Since(begin),
			"error", err,
			"response", responseJSON,
		)
	}(time.Now())
	res = s.Service.Exec(ctx, req)
	return res
}

func (s *loggingService) inBlacklist(operation string) bool {
	return s.blacklist[operation]
}

func findOpName(req string) string {
	var op string
	var other string
	_, err := fmt.Fscanf(strings.NewReader(req), "{ mutation: %s %s", &op, &other)
	if err != nil {
		fmt.Fscanf(strings.NewReader(req), "{ %s %s", &op, &other)
	}
	return strings.Split(op, "(")[0]
}
