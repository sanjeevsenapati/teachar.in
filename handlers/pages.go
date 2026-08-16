package handlers

import (
	"net/http"

	"teachar.in/models"
)

// homeHandler serves the home page.
func (app *Application) homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		app.notFoundError(w, r)
		return
	}

	// For the "Featured Menu" section, we fetch a few items.
	featuredItems, err := app.MenuService.GetFeaturedItems(r.Context())
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	data := models.PageData{
		"Title":         "A Better Tea Experience",
		"FeaturedItems": featuredItems,
	}
	app.render(w, r, http.StatusOK, "home.html", data)
}

// menuHandler serves the main menu page.
func (app *Application) menuHandler(w http.ResponseWriter, r *http.Request) {
	menu, err := app.MenuService.GetFullMenu(r.Context())
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	data := models.PageData{"Title": "Our Menu", "Menu": menu}
	app.render(w, r, http.StatusOK, "menu.html", data)
}

// aboutHandler serves the about page.
func (app *Application) aboutHandler(w http.ResponseWriter, r *http.Request) {
	data := models.PageData{"Title": "About TEACHAR"}
	app.render(w, r, http.StatusOK, "about.html", data)
}
