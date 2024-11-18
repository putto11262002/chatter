package chatter

import "net/http"

func HTMXRedirect(w http.ResponseWriter, r *http.Request, url string, httpCode int) {
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", url)
		return
	}
	http.Redirect(w, r, url, httpCode)
}
