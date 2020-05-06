package api

import (
	"errors"
	"github.com/Bnei-Baruch/auth-api/pkg/httputil"
	"github.com/Bnei-Baruch/auth-api/pkg/middleware"
	"github.com/Nerzal/gocloak/v5"
	"github.com/gorilla/mux"
	pkgerr "github.com/pkg/errors"
	"net/http"
	"strconv"
	"time"
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

	err = errors.New("not found")
	return nil, err
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

	err := errors.New("current user not found")
	return nil, err
}

func (a *App) getRequestedUser(email string) (*gocloak.User, error) {
	user, err := a.getUserByEmail(email)
	if err != nil {
		return nil, err
	}

	// Get User Groups
	var groups []*gocloak.UserGroup
	groups, err = a.client.GetUserGroups(a.token.AccessToken, "main", *user.ID)
	if err != nil {
		return nil, err
	}

	// Make sure requested user in galaxy group
	if len(groups) > 0 {
		for _, g := range groups {
			if *g.ID == "04778f5d-31c1-4a2d-a395-7eac07ebc5b7" {
				return user, nil
			}
		}
	}

	err = errors.New("requested user not found")
	return nil, err
}

func (a *App) setRequest(w http.ResponseWriter, r *http.Request) {

	// Get Current User
	cu, err := a.getCurrentUser(r)
	if err != nil {
		httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
		return
	}

	// Check if request already done
	if _, ok := cu.Attributes["request"]; ok {
		httputil.RespondWithError(w, http.StatusBadRequest, "Request Already Done")
		return
	}

	// Get requested user
	email := r.FormValue("email")
	ru, err := a.getUserByEmail(email)
	if err != nil {
		httputil.RespondWithJSON(w, http.StatusNotFound, err)
		return
	}

	// Set request attributes to current user
	timestamp := int(time.Now().UnixNano() / int64(time.Millisecond))
	if cu.Attributes == nil {
		cu.Attributes = map[string][]string{}
		cu.Attributes["locale"] = []string{"en"}
	}
	cu.Attributes["request"] = []string{email}
	cu.Attributes["timestamp"] = []string{strconv.Itoa(timestamp)}
	err = a.client.UpdateUser(a.token.AccessToken, "main", *cu)
	if err != nil {
		httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
		return
	}

	// Set request attributes to requested user
	if ru.Attributes == nil {
		ru.Attributes = map[string][]string{}
		cu.Attributes["locale"] = []string{"en"}
	}
	if val, ok := ru.Attributes["pending"]; ok {
		val = append(val, *cu.Email)
		ru.Attributes["pending"] = val
	} else {
		ru.Attributes["pending"] = []string{*cu.Email}
	}
	err = a.client.UpdateUser(a.token.AccessToken, "main", *ru)
	if err != nil {
		httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
		return
	}

	httputil.RespondSuccess(w)
}

func (a *App) verifyUser(w http.ResponseWriter, r *http.Request) {

	// Check role
	chk := checkRole("gxy_user", r)
	if !chk {
		e := errors.New("bad permission")
		httputil.NewUnauthorizedError(e).Abort(w, r)
		return
	}

	// Get Pending User by Mail
	email := r.FormValue("email")
	pu, err := a.getUserByEmail(email)
	if err != nil {
		httputil.RespondWithJSON(w, http.StatusNotFound, err)
		return
	}

	// Parse action
	action := r.FormValue("action")
	if action == "ignore" {
		err := a.setVerify("ignore", email, pu, r)
		if err != nil {
			httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
			return
		}

		httputil.RespondSuccess(w)
		return
	}

	// Get Pending User Groups
	var groups []*gocloak.UserGroup
	groups, err = a.client.GetUserGroups(a.token.AccessToken, "main", *pu.ID)
	if err != nil {
		httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
		return
	}

	if len(groups) > 0 {
		for _, g := range groups {

			// Make sure pending user in pending group
			if *g.ID == "c46f3890-fa01-4933-968d-488ba5ca3153" {

				// Change pending User group
				err = a.client.DeleteUserFromGroup(a.token.AccessToken, "main", *pu.ID, "c46f3890-fa01-4933-968d-488ba5ca3153")
				if err != nil {
					httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
					return
				}

				err = a.client.AddUserToGroup(a.token.AccessToken, "main", *pu.ID, "04778f5d-31c1-4a2d-a395-7eac07ebc5b7")
				if err != nil {
					httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
					return
				}

				// Set verify user attribute
				err := a.setVerify("approve", email, pu, r)
				if err != nil {
					//FIXME: does we need rollback group change?
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

func (a *App) setVerify(action string, email string, pu *gocloak.User, r *http.Request) error {
	// Get Current User
	cu, err := a.getCurrentUser(r)
	if err != nil {
		return err
	}

	// Set approved to current user attribute
	if action == "approve" {
		if cu.Attributes == nil {
			cu.Attributes = map[string][]string{}
			cu.Attributes["locale"] = []string{"en"}
		}
		if val, ok := cu.Attributes["approved"]; ok {
			val = append(val, *cu.Email)
			cu.Attributes["approved"] = val
		} else {
			cu.Attributes["approved"] = []string{email}
		}
	}

	// Remove pending attribute from current user
	if cu.Attributes == nil {
		cu.Attributes = map[string][]string{}
		cu.Attributes["locale"] = []string{"en"}
	}
	if val, ok := cu.Attributes["pending"]; ok {
		if len(val) > 1 {
			for i, v := range val {
				if v == email {
					val = append(val[:i], val[i+1:]...)
					break
				}
			}
			cu.Attributes["pending"] = val
		} else {
			delete(cu.Attributes, "pending")
		}
	}
	err = a.client.UpdateUser(a.token.AccessToken, "main", *cu)
	if err != nil {
		return err
	}

	// Remove request attribute from pending user
	if _, ok := pu.Attributes["request"]; ok {
		delete(pu.Attributes, "request")
		delete(pu.Attributes, "timestamp")
	}

	if action == "ignore" {
		// Add User to banned group
		err := a.client.DeleteUserFromGroup(a.token.AccessToken, "main", *pu.ID, "c46f3890-fa01-4933-968d-488ba5ca3153")
		if err != nil {
			return err
		}

		err = a.client.AddUserToGroup(a.token.AccessToken, "main", *pu.ID, "c4569eaa-c67d-446e-b370-ad426a006c6b")
		if err != nil {
			return err
		}
		*pu.Enabled = false
	}

	err = a.client.UpdateUser(a.token.AccessToken, "main", *pu)
	if err != nil {
		return err
	}

	return nil
}

func (a *App) approveUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Check role
	chk := checkRole("gxy_root", r)
	if !chk {
		e := errors.New("bad permission")
		httputil.NewUnauthorizedError(e).Abort(w, r)
		return
	}

	// Change User group
	err := a.client.DeleteUserFromGroup(a.token.AccessToken, "main", id, "c46f3890-fa01-4933-968d-488ba5ca3153")
	if err != nil {
		httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
		return
	}

	err = a.client.AddUserToGroup(a.token.AccessToken, "main", id, "04778f5d-31c1-4a2d-a395-7eac07ebc5b7")
	if err != nil {
		httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
		return
	}

	httputil.RespondSuccess(w)
}
