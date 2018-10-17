package graphqlkit

import (
	"context"

	"github.com/go-kit/kit/endpoint"
)

// GraphqlRequest Common fields of graphql request
type GraphqlRequest struct {
	Query         string                 `json:"query"`
	OperationName string                 `json:"operationName"`
	Variables     map[string]interface{} `json:"variables"`
}

func makeGraphqlEndpoint(s Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GraphqlRequest)
		res := s.Exec(ctx, req)
		return res, nil
	}
}
