package graphqlkit

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/go-kit/kit/auth/jwt"
	graphql "github.com/graph-gophers/graphql-go"
)

// Service Interface that all service has to implement to use graphql service
type Service interface {
	Exec(ctx context.Context, req GraphqlRequest) *graphql.Response
}

type graphqlService struct {
	schema *graphql.Schema
}

// NewService Create a new graphql service, reading and resolving schema
func NewService(schema string, resolver interface{}) Service {
	return &graphqlService{
		schema: getGraphqlSchema(schema, resolver),
	}
}

func (s *graphqlService) Exec(ctx context.Context, req GraphqlRequest) *graphql.Response {

	claims := ctx.Value(jwt.JWTClaimsContextKey)
	fmt.Println(claims)
	return s.schema.Exec(ctx, req.Query, req.OperationName, req.Variables)
}

func getGraphqlSchema(schema string, res interface{}) *graphql.Schema {
	schemaFile, err := ioutil.ReadFile(schema)
	if err != nil {
		panic(err)
	}
	return graphql.MustParseSchema(string(schemaFile), res)
}
