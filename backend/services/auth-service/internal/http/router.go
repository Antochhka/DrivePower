package httpserver

import "net/http"

// Routes aggregates handlers for HTTP server.
type Routes struct {
	Signup http.HandlerFunc
	Login  http.HandlerFunc
	Health http.HandlerFunc
}

// NewRouter wires all HTTP routes.
func NewRouter(routes Routes) http.Handler {
	mux := http.NewServeMux()
	if routes.Signup != nil {
		mux.Handle("/auth/signup", method(http.MethodPost, routes.Signup))
	}
	if routes.Login != nil {
		mux.Handle("/auth/login", method(http.MethodPost, routes.Login))
	}
	if routes.Health != nil {
		mux.Handle("/health", method(http.MethodGet, routes.Health))
	}
	return mux
}

func method(expected string, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != expected {
			w.Header().Set("Allow", expected)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		handler(w, r)
	}
}

