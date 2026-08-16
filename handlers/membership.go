package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"teachar.in/middleware"
	"teachar.in/models"
	"teachar.in/services"
)

func (app *Application) clientMembershipHandler(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)

	tiers := app.MembershipService.GetMembershipTiers()

	var activeSub *models.UserSubscription
	if user != nil {
		activeSub, _ = app.MembershipService.GetUserSubscription(r.Context(), user.ID)
	}

	data := models.PageData{
		"Title":              "Cafe Membership & Coffee Pass Subscriptions",
		"Tiers":              tiers,
		"ActiveSubscription": activeSub,
		"User":               user,
	}
	app.render(w, r, http.StatusOK, "membership.html", data)
}

func (app *Application) apiSubscribeMembershipHandler(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		http.Redirect(w, r, "/login?redirect=/membership", http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	tierID := r.FormValue("tier_id")
	paymentMethod := r.FormValue("payment_method")
	if paymentMethod == "" {
		paymentMethod = "UPI"
	}

	sub, err := app.MembershipService.SubscribeUser(r.Context(), user, tierID, paymentMethod)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	if app.AuditService != nil {
		app.AuditService.LogEvent(r.Context(), user, "MEMBERSHIP_SUBSCRIBED",
			fmt.Sprintf("Subscribed to %s (Txn #%s, Amount ₹%.2f)", sub.TierName, sub.TransactionID, sub.PricePaid),
			services.GetClientIP(r))
	}

	http.Redirect(w, r, "/membership?subscribed=true", http.StatusSeeOther)
}

func (app *Application) apiClaimDailyCupHandler(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		http.Error(w, `{"error":"login required"}`, http.StatusUnauthorized)
		return
	}

	if err := app.MembershipService.ClaimDailyCup(r.Context(), user.ID); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	if app.AuditService != nil {
		app.AuditService.LogEvent(r.Context(), user, "MEMBERSHIP_CUP_CLAIMED",
			"Claimed daily free beverage credit",
			services.GetClientIP(r))
	}

	http.Redirect(w, r, "/membership?claimed=true", http.StatusSeeOther)
}

func (app *Application) adminMembershipsHandler(w http.ResponseWriter, r *http.Request) {
	subs, err := app.MembershipService.GetAllSubscriptions(r.Context())
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	var users []models.User
	if app.AuthService != nil {
		users, _ = app.AuthService.GetAllUsers(r.Context())
	}

	tiers := app.MembershipService.GetMembershipTiers()

	var activeCount int
	var mrr float64
	for _, s := range subs {
		if s.Status == "Active" {
			activeCount++
			mrr += s.PricePaid
		}
	}

	data := models.PageData{
		"Title":               "Subscriber Pass & Membership Register",
		"Subscriptions":       subs,
		"Users":               users,
		"Tiers":               tiers,
		"ActiveSubscribers":   activeCount,
		"MonthlyRecurringMRR": mrr,
	}
	app.render(w, r, http.StatusOK, "admin_memberships.html", data)
}

func (app *Application) adminGrantMembershipHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	userID, _ := strconv.ParseInt(r.FormValue("user_id"), 10, 64)
	tierID := r.FormValue("tier_id")
	months, _ := strconv.Atoi(r.FormValue("duration_months"))
	if months <= 0 {
		months = 1
	}

	targetUser, err := app.AuthService.GetUserByID(r.Context(), userID)
	if err != nil {
		app.badRequestError(w, r, fmt.Errorf("user not found: %w", err))
		return
	}

	actor := middleware.GetUserFromContext(r)
	grantedBy := "Admin"
	if actor != nil {
		grantedBy = actor.Name
	}

	sub, err := app.MembershipService.GrantUserSubscription(r.Context(), targetUser, tierID, months, grantedBy)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	if app.AuditService != nil && actor != nil {
		app.AuditService.LogEvent(r.Context(), actor, "MEMBERSHIP_GRANTED",
			fmt.Sprintf("Granted %s to %s (%s) for %d month(s)", sub.TierName, targetUser.Name, targetUser.Email, months),
			services.GetClientIP(r))
	}

	http.Redirect(w, r, "/admin/memberships?granted=true", http.StatusSeeOther)
}

func (app *Application) adminCancelMembershipHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	id, _ := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err := app.MembershipService.CancelSubscription(r.Context(), id); err != nil {
		app.serverError(w, r, err)
		return
	}

	actor := middleware.GetUserFromContext(r)
	if app.AuditService != nil && actor != nil {
		app.AuditService.LogEvent(r.Context(), actor, "MEMBERSHIP_CANCELLED",
			fmt.Sprintf("Cancelled Subscription #%d", id),
			services.GetClientIP(r))
	}

	http.Redirect(w, r, "/admin/memberships", http.StatusSeeOther)
}
