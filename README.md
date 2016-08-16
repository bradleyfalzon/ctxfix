# Introduction

Go 1.7 brought introduced the `context` package, which can be used instead of `golang.org/x/net/context`.

`ctxfix` assists in migration code to the new package, and was purpose built to solve one problem - it's not feature complete
and will probably not work as intended on your base.

- It does not write correct Go code, it just attempts some common rewrites, this is only due to time constraints
- You will still need to run `goimports` to format the import paths correctly
- Backups of files are not taken, it's assumed your VCS will show you the changes made and allow reverting
- It's likely some decent `sed` will get you further than this tool

This tool was used to migrate to the new `context` package on a couple code bases, but only with the combination of hand editing.

Although I'd like this tool to help everyone, it's unlikely to do so, and I'm unlikely to support further changes as it's
already done its job for me. It's simply made available for others in case it helps them. Discussions welcome, but this won't likely
be a supported tool.

# What it does

- Finds all `.go` files in the local directory, parse using `go/ast` (no type checking)
- Rewrite (in place) import of `golang.org/x/net/context` to `context`
- Find function declarations containing `context.Context` and `*http.Request`, remove `context.Context`
- Within those functions, changes the local variable for `context.Context` (probably `ctx`) and rewrite to `*http.Request`'s variable with its context (e.g. `r.Context()`)

You'll still need to:

- Handle setting of the context values, this only deals with reading of the values
- Run `goimports` to reorder the context import (in hindsight the rewrite is redundant if we're running `goimports` anyway)
- There's a countless edge cases that this tool didn't need to handle, which you may need to

# Example

Example of the changes made, here's an application which stores the `User-Agent` inside context, following the pattern outlined: https://blog.golang.org/error-handling-and-go#TOC_3. as seen in `testdata/in.go`:

```go
package main

import (
        "fmt"
        "net/http"
        "strings"

        "golang.org/x/net/context"
)

func main() {
        http.Handle("/", httpHandler(homeHandler))
        http.ListenAndServe(":3000", nil)
}

type httpHandler func(context.Context, http.ResponseWriter, *http.Request) (int, error)

func (h httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
        ctx := context.Background()

        ua := r.Header.Get("User-Agent")
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
```

`ctxfix` will adjust the import and `homeHandler` signature and body correctly, as seen in `testdata/ctxfix.go`:

```diff
-func homeHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) (int, error) {
-       fmt.Fprint(w, "Hello: ", ctx.Value("ua"))
+func homeHandler(w http.ResponseWriter, r *http.Request) (int, error) {
+       fmt.Fprint(w, "Hello: ", r.Context().Value("ua"))
```

But you are expected to rewrite the `httpHandler` signature and `ServeHTTP` method:

```diff
-type httpHandler func(context.Context, http.ResponseWriter, *http.Request) (int, error)
+type httpHandler func(http.ResponseWriter, *http.Request) (int, error)

 func (h httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
-       ctx := context.Background()
-
        ua := r.Header.Get("User-Agent")
-       ctx = context.WithValue(ctx, "ua", ua)
+       ctx := context.WithValue(r.Context(), "ua", ua)
+       r = r.WithContext(ctx)

-       code, err := h(ctx, w, r)
+       code, err := h(w, r)
        if err != nil {
                http.Error(w, http.StatusText(code), code)
        }
 }
```

For after `ctxfix` and hand editing, as seen in `testdata/final.go`:

```go
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
```
