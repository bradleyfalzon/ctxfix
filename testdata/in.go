package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"golang.org/x/net/context"
)

func main() {
	http.Handle("/", httpHandler(homeHandler))
	log.Fatal(http.ListenAndServe(":3000", nil))
}

type httpHandler func(context.Context, http.ResponseWriter, *http.Request) (int, error)

func (h httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	ua := strings.ToLower(r.Header.Get("User-Agent"))
	ctx = context.WithValue(ctx, "ua", ua)

	code, err := h(ctx, w, r)
	if err != nil {
		http.Error(w, http.StatusText(code), code)
	}
}

func homeHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	fmt.Fprint(w, "Hello: ", ctx.Value("ua"))
	return http.StatusOK, nil
}
