package graphqlkit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	jwt "github.com/dgrijalva/jwt-go"
	fields "github.com/gbaptista/requested-fields"
	gokitjwt "github.com/go-kit/kit/auth/jwt"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	httptransport "github.com/go-kit/kit/transport/http"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

type authentication struct {
	key    []byte
	method *jwt.SigningMethodHMAC
	claims gokitjwt.ClaimsFactory
}

type instrumenting struct {
	namespace string
	subsystem string
}

// Handlers Take care of all possible service added with graphql endpoint
type Handlers struct {
	service Service
	authentication
	logger log.Logger
	instrumenting
	options       []httptransport.ServerOption
	logBlacklist  []string
	authBlacklist []string
}

// AddGraphqlService Create a new Service graphql and add to handler
func (h *Handlers) AddGraphqlService(schema string, resolver interface{}) {
	h.service = NewService(schema, resolver)
}

// AddLoggingService Add logging Service to handler
func (h *Handlers) AddLoggingService(logger log.Logger) {
	h.logger = logger
}

// AddInstrumentingService Add Instrumenting Service to handler
func (h *Handlers) AddInstrumentingService(namespace, moduleName string) {
	h.namespace = namespace
	h.subsystem = moduleName
}

// AddAuthenticationService Add Authentication Service to handler
func (h *Handlers) AddAuthenticationService(
	secret string,
	method *jwt.SigningMethodHMAC,
	claims jwt.Claims) {

	h.key = []byte(secret)
	h.method = method
	h.claims = func() jwt.Claims {
		return claims
	}
}

// AddFullGraphqlService Add all service available
func (h *Handlers) AddFullGraphqlService(
	schema string,
	resolver interface{},
	logger log.Logger,
	namespace, moduleName, secret string,
	method *jwt.SigningMethodHMAC,
	claims jwt.Claims,
) {
	h.AddGraphqlService(schema, resolver)
	h.AddLoggingService(logger)
	h.AddInstrumentingService(namespace, moduleName)
	h.AddAuthenticationService(secret, method, claims)
}

// AddLoggingBlacklist Add a method for not be logging
func (h *Handlers) AddLoggingBlacklist(methods []string) {
	h.logBlacklist = append(h.logBlacklist, methods...)
}

// AddAuthBlacklist Add a method for not be logging
func (h *Handlers) AddAuthBlacklist(methods []string) {
	h.authBlacklist = append(h.authBlacklist, methods...)
}

// AddServerOptions Add server options to handler
func (h *Handlers) AddServerOptions(options ...httptransport.ServerOption) {
	h.options = append(h.options, options...)
}

// Handler Retorns the http handler with all services added
func (h *Handlers) Handler() http.Handler {
	h.addLogging()
	h.addInstrumenting()
	var httpEndpoint endpoint.Endpoint
	if h.authenticationEnabled() {
		httpEndpoint = h.getEndpointWithAuthentication()
	} else {
		httpEndpoint = makeGraphqlEndpoint(h.service)
	}
	h.AddServerOptions(httptransport.ServerBefore(fieldsToCtx()))

	return httptransport.NewServer(
		httpEndpoint,
		decodeGraphqlRequest,
		encodeResponse,
		h.options...,
	)
}

func fieldsToCtx() httptransport.RequestFunc {
	return func(ctx context.Context, r *http.Request) context.Context {
		var params GraphqlRequest
		bodyBytes, _ := ioutil.ReadAll(r.Body)
		if err := json.Unmarshal(bodyBytes, &params); err != nil {
			fmt.Print(err)
			return nil
		}
		r.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

		return context.WithValue(ctx,
			fields.ContextKey, fields.BuildTree(params.Query, params.Variables))
	}
}

func (h *Handlers) addLogging() {
	if h.logger != nil {
		h.options = append(h.options,
			httptransport.ServerErrorLogger(h.logger),
		)
		h.service = NewLoggingService(h.logger, h.service, h.logBlacklist)
	}
}

func (h *Handlers) addInstrumenting() {
	if h.namespace != "" {
		requestCount := kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: h.namespace,
			Subsystem: h.subsystem,
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, fieldKeys)
		requestLatency := kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: h.namespace,
			Subsystem: h.subsystem,
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, fieldKeys)

		h.service = NewInstrumentingService(requestCount, requestLatency, h.service)
	}
}

func (h *Handlers) authenticationEnabled() bool {
	return h.key != nil
}

func (h *Handlers) getEndpointWithAuthentication() endpoint.Endpoint {
	h.options = append(h.options,
		httptransport.ServerErrorEncoder(authErrorEncoder),
		httptransport.ServerBefore(gokitjwt.HTTPToContext()))
	auth := JwtEndpoint{
		func(token *jwt.Token) (interface{}, error) {
			return h.key, nil
		},
		h.method,
		h.claims,
	}
	end := makeGraphqlEndpoint(h.service)

	return makeBlacklistMiddleware(auth.NewParser(end), h.authBlacklist)(end)
}
