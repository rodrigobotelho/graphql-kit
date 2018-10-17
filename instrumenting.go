package graphqlkit

import (
	"context"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	gokitjwt "github.com/go-kit/kit/auth/jwt"

	"github.com/go-kit/kit/metrics"

	graphql "github.com/graph-gophers/graphql-go"
)

type instrumentingService struct {
	requestCount   metrics.Counter
	requestLatency metrics.Histogram
	Service
}

var fieldKeys = []string{"method", "client", "query"}

// NewInstrumentingService returns an instance of an instrumenting Service.
func NewInstrumentingService(counter metrics.Counter, latency metrics.Histogram, s Service) Service {
	return &instrumentingService{
		requestCount:   counter,
		requestLatency: latency,
		Service:        s,
	}
}

func (s *instrumentingService) Exec(ctx context.Context, req GraphqlRequest) (res *graphql.Response) {
	defer func(begin time.Time) {
		standardCl, converted := ctx.Value(gokitjwt.JWTClaimsContextKey).(*jwt.StandardClaims)
		var subject string
		if !converted {
			subject = "Not Authenticated"
		} else {
			subject = standardCl.Subject
		}
		lvs := []string{
			"method", req.OperationName,
			"client", subject,
			"query", req.Query}
		s.requestCount.With(lvs...).Add(1)
		s.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	return s.Service.Exec(ctx, req)
}
