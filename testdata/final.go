package main

import (
	"fmt"
	"net/http"

	"context"
)

func main() {
	http.Handle("/", httpHandler(homeHandler))
	http.ListenAndServe(":3000", nil)
}

type httpHandler func(http.ResponseWriter, *http.Request) (int, error)

func (h httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ua := r.Header.Get("User-Agent")
	ctx := context.WithValue(r.Context(), "ua", ua)
	r = r.WithContext(ctx)

	code, err := h(w, r)
	if err != nil {
		http.Error(w, http.StatusText(code), code)
	}
}

func homeHandler(w http.ResponseWriter, r *http.Request) (int, error) {
	fmt.Fprint(w, "Hello: ", r.Context().Value("ua"))
	return http.StatusOK, nil
}
