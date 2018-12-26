package graphqlkit

import (
	"context"
	"regexp"
	"strings"

	"github.com/go-kit/kit/endpoint"
)

func makeBlacklistMiddleware(end endpoint.Endpoint, blacklist []string) endpoint.Middleware {
	bl := make(map[string]bool)
	for _, method := range blacklist {
		bl[strings.ToUpper(method)] = true
	}
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			req := request.(GraphqlRequest)
			if req.OperationName == "" {
				req.OperationName = findOpName(req.Query)
			}
			if bl[strings.ToUpper(req.OperationName)] {
				return next(ctx, request)
			}
			return end(ctx, request)
		}
	}
}

func findOpName(req string) string {
	findOpAfterBracesWithOrWithoutSpace := "{([\t\n\v\f\r ]*)([0-9A-Za-z_]+)"
	r := regexp.MustCompile(findOpAfterBracesWithOrWithoutSpace)
	sm := r.FindStringSubmatch(req)
	str := ""
	if len(sm) > 0 {
		str = sm[len(sm)-1]
	}
	foundWithSpace := strings.Split(str, " ")
	if len(foundWithSpace) > 1 {
		return foundWithSpace[1]
	}
	foundWithoutSpace := foundWithSpace[0]
	return strings.Replace(foundWithoutSpace, "{", "", -1)
}
