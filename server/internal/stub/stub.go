package stub

import "net/http"

var notImplementedBody = []byte(`{"error":"not implemented"}`)

// NotImplemented writes a 501 JSON response.
// Use this in handler functions that have not yet been transcribed from the Kotlin source.
func NotImplemented(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	_, _ = w.Write(notImplementedBody)
}
