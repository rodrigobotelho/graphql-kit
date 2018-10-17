### Project to use go-kit with graph-gophers/graphql-go ###

This project has the objective to use the facilities of go-kit
together with the facilities of graph-gophers/graphql-go.
    
It creates an api to add services as graphql, logging, instrumenting
and authenticating.

[Go kit](https://github.com/go-kit/kit)  
[graphql-go](https://github.com/graph-gophers/graphql-go)  

### Example of utilization ###
```
h := graphql-kit.Handlers{}
h.AddGraphqlService(schema, resolver)
h.AddLoggingService(logger)
h.AddInstrumentingService(namespace, moduleName)
h.AddAuthenticationService(secret, method, claims)
http.Handle("/graphql", h.Handler())
```
### Another option ###
h := graphql-kit.Handlers{}
h.AddFullGraphqlService(
  schema, resolver,
  logger,
  namespace, moduleName,
  secret, method, claims
)
http.Handle("/graphql", h.Handler())
```
