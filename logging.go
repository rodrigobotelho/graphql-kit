package graphqlkit

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kit/kit/log"
	graphql "github.com/graph-gophers/graphql-go"
)

type loggingService struct {
	logger log.Logger
	Service
}

// NewLoggingService Create a logging service that logs method, query,
// how much it took and possible erros
func NewLoggingService(logger log.Logger, s Service) Service {
	return &loggingService{logger, s}
}

func (s *loggingService) Exec(ctx context.Context, req GraphqlRequest) (res *graphql.Response) {
	defer func(begin time.Time) {
		var err error
		if len(res.Errors) > 0 {
			err = fmt.Errorf("request error: %v", res.Errors)
		}
		s.logger.Log(
			"method", req.OperationName,
			"query", req.Query,
			"took", time.Since(begin),
			"error", err,
		)
	}(time.Now())
	res = s.Service.Exec(ctx, req)
	return res
}
