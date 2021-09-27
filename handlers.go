package graphqlkit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	fields "github.com/gbaptista/requested-fields"
	gokitjwt "github.com/go-kit/kit/auth/jwt"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	httptransport "github.com/go-kit/kit/transport/http"
	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

type contextKey string

const (
	SchemaKey  contextKey = "schema"
	RequestKey contextKey = "request"
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
	options               []httptransport.ServerOption
	logBlacklist          []string
	logFullBlacklist      []string
	logVariablesBlacklist map[string][]string
	authBlacklist         []string
	schemaString          string
}

// AddGraphqlService Create a new Service graphql and add to handler
func (h *Handlers) AddGraphqlService(schema string, resolver interface{}) {
	h.service, h.schemaString = NewService(schema, resolver)
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
	claimsFactory gokitjwt.ClaimsFactory) {

	h.key = []byte(secret)
	h.method = method
	h.claims = claimsFactory
}

// AddFullGraphqlService Add all service available
func (h *Handlers) AddFullGraphqlService(
	schema string,
	resolver interface{},
	logger log.Logger,
	namespace, moduleName, secret string,
	method *jwt.SigningMethodHMAC,
	claimsFactory gokitjwt.ClaimsFactory,
) {
	h.AddGraphqlService(schema, resolver)
	h.AddLoggingService(logger)
	h.AddInstrumentingService(namespace, moduleName)
	h.AddAuthenticationService(secret, method, claimsFactory)
}

// AddLoggingBlacklist Add a method for not be logging
func (h *Handlers) AddLoggingBlacklist(methods []string) {
	h.logBlacklist = append(h.logBlacklist, methods...)
}

// AddLoggingFullBlacklist Add a method for not be logging anyway with or without erros
func (h *Handlers) AddLoggingFullBlacklist(methods []string) {
	h.logFullBlacklist = append(h.logFullBlacklist, methods...)
}

// AddLoggingVariablesBlacklist Add a variables list of a method for not be logging
func (h *Handlers) AddLoggingVariablesBlacklist(methodsvariables map[string][]string) {
	if h.logVariablesBlacklist == nil {
		h.logVariablesBlacklist = make(map[string][]string)
	}
	for method, variables := range methodsvariables {
		h.logVariablesBlacklist[method] = append(h.logVariablesBlacklist[method], variables...)
	}
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
	h.AddServerOptions(httptransport.ServerBefore(schemaToCtx(h.schemaString)))
	h.AddServerOptions(httptransport.ServerBefore(requestToCtx()))
	h.AddServerOptions(httptransport.ServerBefore(httptransport.PopulateRequestContext))
	h.AddServerOptions(httptransport.ServerBefore(requestIdToCtx()))

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
			return ctx
		}
		r.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

		return context.WithValue(ctx,
			fields.ContextKey, fields.BuildTree(params.Query, params.Variables))
	}
}

func requestToCtx() httptransport.RequestFunc {
	return func(ctx context.Context, r *http.Request) context.Context {
		bodyBytes, _ := ioutil.ReadAll(r.Body)
		r.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
		return context.WithValue(ctx, RequestKey, bodyBytes)
	}
}

func schemaToCtx(schemaString string) httptransport.RequestFunc {
	return func(ctx context.Context, r *http.Request) context.Context {
		return context.WithValue(ctx, SchemaKey, schemaString)
	}
}

func requestIdToCtx() httptransport.RequestFunc {
	return func(ctx context.Context, r *http.Request) context.Context {
		reqId, ok := ctx.Value(httptransport.ContextKeyRequestXRequestID).(string)
		if !ok || reqId == "" {
			if reqId := r.Header.Get("X-Request-Id"); reqId != "" {
				return context.WithValue(ctx, httptransport.ContextKeyRequestXRequestID, reqId)
			}
			reqId, err := uuid.NewRandom()
			if err == nil {
				return context.WithValue(ctx, httptransport.ContextKeyRequestXRequestID, reqId.String())
			}
		}
		return ctx
	}
}

func (h *Handlers) addLogging() {
	if h.logger != nil {
		h.options = append(h.options,
			httptransport.ServerErrorLogger(h.logger),
		)
		h.service = NewLoggingService(h.logger, h.service, h.logBlacklist, h.logFullBlacklist, h.logVariablesBlacklist)
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
	end := makeGraphqlEndpoint(h.service)

	return makeBlacklistMiddleware(MakeAuthenticationEndPoint(end, h.key, h.method, h.claims), h.authBlacklist)(end)
}
