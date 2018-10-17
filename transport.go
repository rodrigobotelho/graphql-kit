package graphqlkit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

var errBadRequest = errors.New("bad request")

func decodeGraphqlRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var params GraphqlRequest
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		fmt.Print(err)
		return nil, errBadRequest
	}

	return params, nil
}

func encodeResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	responseJSON, err := json.Marshal(response)
	if err != nil {
		encodeError(ctx, err, w)
		return nil
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(responseJSON)
	return err
}

type errorer interface {
	error() error
}

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": err.Error(),
	})
}

type authResponse struct {
	Token string `json:"token,omitempty"`
	Err   string `json:"err,omitempty"`
}

func authErrorEncoder(_ context.Context, err error, w http.ResponseWriter) {
	code := http.StatusUnauthorized
	msg := err.Error()

	w.WriteHeader(code)
	json.NewEncoder(w).Encode(authResponse{Token: "", Err: msg})
}
