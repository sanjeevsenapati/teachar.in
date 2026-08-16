package handlers

import (
	"fmt"
	"net/http"
)

// serverError logs the detailed error and sends a generic 500 response.
func (app *Application) serverError(w http.ResponseWriter, r *http.Request, err error) {
	app.Logger.Error(err.Error(), "method", r.Method, "uri", r.URL.RequestURI())

	// For API requests, send JSON. Otherwise, render an HTML error page.
	if r != nil && r.Header.Get("Accept") == "application/json" {
		app.errorJSON(w, r, http.StatusInternalServerError, "the server encountered a problem and could not process your request")
	} else {
		message := "Sorry, something went wrong on our end."
		app.render(w, r, http.StatusInternalServerError, "error.html", map[string]interface{}{"Title": "Server Error", "Message": message})
	}
}

// notFoundError sends a 404 Not Found response.
func (app *Application) notFoundError(w http.ResponseWriter, r *http.Request) {
	if r != nil && r.Header.Get("Accept") == "application/json" {
		app.errorJSON(w, r, http.StatusNotFound, "the requested resource could not be found")
	} else {
		message := "We can't find the page you're looking for."
		app.render(w, r, http.StatusNotFound, "error.html", map[string]interface{}{"Title": "Page Not Found", "Message": message})
	}
}

// methodNotAllowedError sends a 405 Method Not Allowed response.
func (app *Application) methodNotAllowedError(w http.ResponseWriter, r *http.Request) {
	message := fmt.Sprintf("The %s method is not supported for this resource.", r.Method)
	app.errorJSON(w, r, http.StatusMethodNotAllowed, message)
}

// badRequestError sends a 400 Bad Request response.
func (app *Application) badRequestError(w http.ResponseWriter, r *http.Request, err error) {
	app.errorJSON(w, r, http.StatusBadRequest, err.Error())
}
