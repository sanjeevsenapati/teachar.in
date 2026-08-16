package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"teachar.in/middleware"
	"teachar.in/models"
	"teachar.in/services"
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

	if app.AuditService != nil {
		app.AuditService.LogEvent(r.Context(), user, "USER_LOGIN", "User logged in successfully.", services.GetClientIP(r))
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    session.ID,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	if user.Role == "admin" || user.Role == "superadmin" || user.Role == "staff" {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}

	if redirect != "" {
		http.Redirect(w, r, redirect, http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
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

	if app.AuditService != nil {
		app.AuditService.LogEvent(r.Context(), user, "USER_REGISTER", "Client registered new account: "+user.Email, services.GetClientIP(r))
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    session.ID,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
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

func (app *Application) apiUpdateProfileHandler(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Parse multipart form for avatar image uploads (up to 10MB)
	_ = r.ParseMultipartForm(10 << 20)

	name := r.FormValue("name")
	email := r.FormValue("email")
	mobile := r.FormValue("mobile_number")
	address := r.FormValue("address")
	avatarURL := r.FormValue("avatar")

	// Check if custom avatar file was uploaded
	file, header, err := r.FormFile("avatar_file")
	if err == nil && file != nil {
		defer file.Close()

		avatarDir := "./static/images/avatars"
		_ = os.MkdirAll(avatarDir, 0755)

		ext := filepath.Ext(header.Filename)
		if ext == "" {
			ext = ".jpg"
		}
		avatarPath := filepath.Join(avatarDir, fmt.Sprintf("user_%d%s", user.ID, ext))

		out, createErr := os.Create(avatarPath)
		if createErr == nil {
			defer out.Close()
			_, _ = io.Copy(out, file)
			avatarURL = fmt.Sprintf("/static/images/avatars/user_%d%s", user.ID, ext)
		}
	}

	updatedUser, err := app.AuthService.UpdateUserProfile(r.Context(), user.ID, name, email, mobile, address, avatarURL)
	if err != nil {
		data := models.PageData{
			"Title": "My Profile Account",
			"User":  user,
			"Error": err.Error(),
		}
		app.render(w, r, http.StatusBadRequest, "client_account.html", data)
		return
	}

	if app.AuditService != nil {
		app.AuditService.LogEvent(r.Context(), updatedUser, "USER_PROFILE_UPDATED", "Updated profile details", services.GetClientIP(r))
	}

	http.Redirect(w, r, "/account?updated=true", http.StatusSeeOther)
}
