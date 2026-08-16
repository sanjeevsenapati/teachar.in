package handlers

import (
	"net/http"
	"time"

	"teachar.in/models"
)

func (app *Application) loginPageHandler(w http.ResponseWriter, r *http.Request) {
	redirect := r.URL.Query().Get("redirect")
	data := models.PageData{
		"Title":    "Sign In",
		"Redirect": redirect,
	}
	app.render(w, r, http.StatusOK, "login.html", data)
}

func (app *Application) loginSubmitHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")
	redirect := r.FormValue("redirect")

	user, err := app.AuthService.AuthenticateUser(r.Context(), email, password)
	if err != nil {
		data := models.PageData{
			"Title":    "Sign In",
			"Error":    "Invalid email or password. Please try again.",
			"Email":    email,
			"Redirect": redirect,
		}
		app.render(w, r, http.StatusUnauthorized, "login.html", data)
		return
	}

	session, err := app.AuthService.CreateSession(r.Context(), user.ID)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    session.ID,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	if user.Role == "admin" {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}

	if redirect != "" {
		http.Redirect(w, r, redirect, http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/menu", http.StatusSeeOther)
}

func (app *Application) registerPageHandler(w http.ResponseWriter, r *http.Request) {
	data := models.PageData{"Title": "Create Account"}
	app.render(w, r, http.StatusOK, "register.html", data)
}

func (app *Application) registerSubmitHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	name := r.FormValue("name")
	email := r.FormValue("email")
	mobile := r.FormValue("mobile")
	password := r.FormValue("password")

	user, err := app.AuthService.RegisterUser(r.Context(), name, email, mobile, password)
	if err != nil {
		data := models.PageData{
			"Title":  "Create Account",
			"Error":  err.Error(),
			"Name":   name,
			"Email":  email,
			"Mobile": mobile,
		}
		app.render(w, r, http.StatusBadRequest, "register.html", data)
		return
	}

	session, err := app.AuthService.CreateSession(r.Context(), user.ID)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    session.ID,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/menu", http.StatusSeeOther)
}

func (app *Application) logoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err == nil && cookie.Value != "" {
		app.AuthService.Logout(r.Context(), cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
