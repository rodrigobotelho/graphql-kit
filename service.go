package graphqlkit

import (
	"context"
	"io/ioutil"

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
func NewService(schemaFilename string, resolver interface{}) (Service, string) {
	schema, schemaString := getGraphqlSchema(schemaFilename, resolver)
	return &graphqlService{schema}, schemaString
}

func (s *graphqlService) Exec(ctx context.Context, req GraphqlRequest) *graphql.Response {
	return s.schema.Exec(ctx, req.Query, req.OperationName, req.Variables)
}

func getGraphqlSchema(schemaFilename string, res interface{}) (*graphql.Schema, string) {
	schemaBytes, err := ioutil.ReadFile(schemaFilename)
	if err != nil {
		panic(err)
	}
	opts := []graphql.SchemaOpt{graphql.UseFieldResolvers(), graphql.UseStringDescriptions()}
	schemaString := string(schemaBytes)
	return graphql.MustParseSchema(schemaString, res, opts...), schemaString
}
