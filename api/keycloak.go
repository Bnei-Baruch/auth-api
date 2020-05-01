package api

import (
	"errors"
	"github.com/Bnei-Baruch/auth-api/pkg/httputil"
	"github.com/Bnei-Baruch/auth-api/pkg/middleware"
	"github.com/Nerzal/gocloak/v5"
	"github.com/gorilla/mux"
	pkgerr "github.com/pkg/errors"
	"net/http"
)

func (a *App) getGroups(w http.ResponseWriter, r *http.Request) {
	g, err := a.client.GetGroups(a.token.AccessToken, "main", gocloak.GetGroupsParams{})
	if err != nil {
		httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
		return
	}

	httputil.RespondWithJSON(w, http.StatusOK, g)
}

func (a *App) getUsers(w http.ResponseWriter, r *http.Request) {
	max := 10000
	params := gocloak.GetUsersParams{Max: &max}
	u, err := a.client.GetUsers(a.token.AccessToken, "main", params)
	if err != nil {
		httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
		return
	}

	httputil.RespondWithJSON(w, http.StatusOK, u)
}

func (a *App) getGroupUsers(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	max := 10000
	params := gocloak.GetGroupsParams{Max: &max}
	g, err := a.client.GetGroupMembers(a.token.AccessToken, "main", id, params)
	if err != nil {
		httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
		return
	}

	httputil.RespondWithJSON(w, http.StatusOK, g)
}

func (a *App) getUserByEmail(email string) (*gocloak.User, error) {
	params := gocloak.GetUsersParams{Email: &email}
	var users []*gocloak.User
	users, err := a.client.GetUsers(a.token.AccessToken, "main", params)
	if err != nil {
		return nil, err
	}

	if len(users) > 0 {
		for _, u := range users {
			if *u.Email == email {
				return u, nil
			}
		}
	}

	return nil, nil
}

func checkRole(role string, r *http.Request) bool {
	if rCtx, ok := middleware.ContextFromRequest(r); ok {
		if rCtx.IDClaims != nil {
			for _, r := range rCtx.IDClaims.RealmAccess.Roles {
				if r == role {
					return true
				}
			}
		}
	}
	return false
}

func (a *App) getCurrentUser(r *http.Request) (*gocloak.User, error) {
	if rCtx, ok := middleware.ContextFromRequest(r); ok {
		if rCtx.IDClaims != nil {
			var user *gocloak.User
			user, err := a.client.GetUserByID(a.token.AccessToken, "main", rCtx.IDClaims.Sub)
			if err != nil {
				return nil, err
			}
			return user, nil
		}
	}
	return nil, nil
}

func (a *App) setRequest(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")

	// Get Current User
	cu, err := a.getCurrentUser(r)
	if err != nil {
		httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
		return
	}

	// Set request attribute
	cu.Attributes["request"] = []string{email}
	err = a.client.UpdateUser(a.token.AccessToken, "main", *cu)
	if err != nil {
		httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
		return
	}

	httputil.RespondSuccess(w)
}

func (a *App) checkUser(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")

	user, err := a.getUserByEmail(email)
	if err != nil {
		httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
		return
	}

	if user == nil {
		httputil.RespondWithJSON(w, http.StatusNotFound, map[string]string{})
		return
	}

	// Get User Groups
	var groups []*gocloak.UserGroup
	groups, err = a.client.GetUserGroups(a.token.AccessToken, "main", *user.ID)
	if err != nil {
		httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
		return
	}

	if len(groups) > 0 {
		for _, g := range groups {

			// Make sure user in galaxy group
			if *g.ID == "04778f5d-31c1-4a2d-a395-7eac07ebc5b7" {
				httputil.RespondWithJSON(w, http.StatusOK, user)
				return
			}
		}
	} else {
		httputil.RespondWithJSON(w, http.StatusNotFound, map[string]string{})
		return
	}

	httputil.RespondWithJSON(w, http.StatusNotFound, map[string]string{})
}

func (a *App) verifyUser(w http.ResponseWriter, r *http.Request) {

	// Check role
	chk := checkRole("gxy_user", r)
	if !chk {
		e := errors.New("bad permission")
		httputil.NewUnauthorizedError(e).Abort(w, r)
		return
	}

	// Get User by Mail
	email := r.FormValue("email")
	user, err := a.getUserByEmail(email)
	if err != nil {
		httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
		return
	}

	if user == nil {
		httputil.RespondWithJSON(w, http.StatusNotFound, map[string]string{})
		return
	}

	// Get User Groups
	var groups []*gocloak.UserGroup
	groups, err = a.client.GetUserGroups(a.token.AccessToken, "main", *user.ID)
	if err != nil {
		httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
		return
	}

	if len(groups) > 0 {
		for _, g := range groups {

			// Make sure user in pending group
			if *g.ID == "c46f3890-fa01-4933-968d-488ba5ca3153" {

				// Change User group
				err = a.client.DeleteUserFromGroup(a.token.AccessToken, "main", *user.ID, "c46f3890-fa01-4933-968d-488ba5ca3153")
				if err != nil {
					httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
					return
				}

				err = a.client.AddUserToGroup(a.token.AccessToken, "main", *user.ID, "04778f5d-31c1-4a2d-a395-7eac07ebc5b7")
				if err != nil {
					httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
					return
				}

				httputil.RespondSuccess(w)
				return
			}
		}
	} else {
		httputil.RespondWithError(w, http.StatusNotFound, "No group found")
		return
	}

	httputil.RespondWithError(w, http.StatusNotFound, "Not in pending group")
}
