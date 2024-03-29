package api

import (
	"context"
	"errors"
	"github.com/Bnei-Baruch/auth-api/pkg/httputil"
	"github.com/Bnei-Baruch/auth-api/pkg/middleware"
	"github.com/Nerzal/gocloak/v11"
	"github.com/gorilla/mux"
	pkgerr "github.com/pkg/errors"
	"net/http"
	"strconv"
	"time"
)

const (
	NewUsers      = "85092f0c-19f7-4963-8a27-adf2fae47dc0"
	GalaxyPending = "c42addf9-5ef6-474c-b5ef-ccc07179c97e"
	GalaxyGuests  = "e1617f1a-ab58-4981-b087-9b997726b821"
	GalaxyUsers   = "04778f5d-31c1-4a2d-a395-7eac07ebc5b7"
	BannedUsers   = "c4569eaa-c67d-446e-b370-ad426a006c6b"
	KenesOlami    = "38275a65-46e4-4294-817b-031e4c07bf2e"
	KmediaUser    = "39211f7f-18e8-4dfa-85c8-82ccbdc9260a"
)

type UserAPI struct {
	//ID               *string                                    `json:"id,omitempty"`
	//CreatedTimestamp *int64                                     `json:"createdTimestamp,omitempty"`
	//Username         *string                                    `json:"username,omitempty"`
	//Enabled          *bool                                      `json:"enabled,omitempty"`
	//EmailVerified    *bool                                      `json:"emailVerified,omitempty"`
	//FirstName        *string                                    `json:"firstName,omitempty"`
	//LastName         *string                                    `json:"lastName,omitempty"`
	//Email            *string                                    `json:"email,omitempty"`
	//Attributes       *map[string][]string                       `json:"attributes,omitempty"`
	Social []*gocloak.FederatedIdentityRepresentation `json:"social,omitempty"`
	Groups []*gocloak.Group                           `json:"groups,omitempty"`
	Roles  []*gocloak.Role                            `json:"roles,omitempty"`
	Cred   []*gocloak.CredentialRepresentation        `json:"credentials,omitempty"`
}

func (a *App) getGroups(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	g, err := a.client.GetGroups(ctx, a.token.AccessToken, "main", gocloak.GetGroupsParams{})
	if err != nil {
		httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
		return
	}

	httputil.RespondWithJSON(w, http.StatusOK, g)
}

func (a *App) getUsers(w http.ResponseWriter, r *http.Request) {
	value := false
	params := gocloak.GetUsersParams{BriefRepresentation: &value}
	ctx := context.Background()
	g, err := a.client.GetUsers(ctx, a.token.AccessToken, "main", params)
	if err != nil {
		httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
		return
	}

	httputil.RespondWithJSON(w, http.StatusOK, g)
}

func (a *App) getVerifyUsers(w http.ResponseWriter, r *http.Request) {

	max := 100000
	params := gocloak.GetUsersParams{Max: &max}
	ctx := context.Background()
	u, err := a.client.GetUsers(ctx, a.token.AccessToken, "main", params)
	if err != nil {
		httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
		return
	}

	var ar []*gocloak.User
	for _, v := range u {
		attr := *v.Attributes
		if _, b := attr["approved"]; b {
			ar = append(ar, v)
		}
	}

	httputil.RespondWithJSON(w, http.StatusOK, ar)
}

func (a *App) getMyInfo(w http.ResponseWriter, r *http.Request) {

	u, err := a.getCurrentUser(r)
	if err != nil {
		httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
		return
	}

	httputil.RespondWithJSON(w, http.StatusOK, u)
}

func (a *App) getGroupUsers(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	first, err := strconv.Atoi(r.FormValue("first"))
	if err != nil {
		first = 0
	}
	max, err := strconv.Atoi(r.FormValue("max"))
	if err != nil {
		max = 15
	}
	search := r.FormValue("search")
	br := true

	params := gocloak.GetGroupsParams{Max: &max, First: &first, Search: &search, BriefRepresentation: &br}
	ctx := context.Background()
	g, err := a.client.GetGroupMembers(ctx, a.token.AccessToken, "main", id, params)
	if err != nil {
		httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
		return
	}

	httputil.RespondWithJSON(w, http.StatusOK, g)
}

func (a *App) findUser(w http.ResponseWriter, r *http.Request) {
	// Check role
	authAdmin := checkRole("auth_admin", r)
	if !authAdmin {
		e := errors.New("bad permission")
		httputil.NewUnauthorizedError(e).Abort(w, r)
		return
	}

	user := &gocloak.User{}
	err := errors.New("error")
	email := r.FormValue("email")
	id := r.FormValue("id")

	if email == "" && id == "" {
		httputil.RespondWithError(w, http.StatusBadRequest, "Params not found")
		return
	}

	if email != "" {
		user, err = a.getUserByEmail(email)
		if err != nil {
			httputil.RespondWithJSON(w, http.StatusNotFound, err)
			return
		}
	}

	if id != "" {
		user, err = a.getUserByID(id)
		if err != nil {
			httputil.RespondWithJSON(w, http.StatusNotFound, err)
			return
		}
	}

	httputil.RespondWithJSON(w, http.StatusOK, user)
}

func (a *App) getUserInfo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	user_id := vars["id"]

	ctx := context.Background()
	var groups []*gocloak.Group
	params := gocloak.GetGroupsParams{}
	groups, _ = a.client.GetUserGroups(ctx, a.token.AccessToken, "main", user_id, params)
	iden, _ := a.client.GetUserFederatedIdentities(ctx, a.token.AccessToken, "main", user_id)
	roles, _ := a.client.GetCompositeRealmRolesByUserID(ctx, a.token.AccessToken, "main", user_id)
	cred, _ := a.client.GetCredentials(ctx, a.token.AccessToken, "main", user_id)

	user_info := &UserAPI{
		iden,
		groups,
		roles,
		cred,
	}

	httputil.RespondWithJSON(w, http.StatusOK, user_info)
}

func (a *App) searchUsers(w http.ResponseWriter, r *http.Request) {
	// Check role
	//authAdmin := checkRole("auth_admin", r)
	//if !authAdmin {
	//	e := errors.New("bad permission")
	//	httputil.NewUnauthorizedError(e).Abort(w, r)
	//	return
	//}

	first, err := strconv.Atoi(r.FormValue("first"))
	if err != nil {
		first = 0
	}
	max, err := strconv.Atoi(r.FormValue("max"))
	if err != nil {
		max = 15
	}
	search := r.FormValue("search")
	br := true

	params := gocloak.GetUsersParams{Max: &max, First: &first, Search: &search, BriefRepresentation: &br}
	ctx := context.Background()
	users, err := a.client.GetUsers(ctx, a.token.AccessToken, "main", params)
	if err != nil {
		httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
		return
	}

	httputil.RespondWithJSON(w, http.StatusOK, users)
}

func (a *App) getUserByID(id string) (*gocloak.User, error) {
	var user *gocloak.User
	ctx := context.Background()
	user, err := a.client.GetUserByID(ctx, a.token.AccessToken, "main", id)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (a *App) getUserByEmail(email string) (*gocloak.User, error) {
	params := gocloak.GetUsersParams{Email: &email}
	var users []*gocloak.User
	ctx := context.Background()
	users, err := a.client.GetUsers(ctx, a.token.AccessToken, "main", params)
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

func (a *App) regCheck(w http.ResponseWriter, r *http.Request) {

	user := &gocloak.User{}
	err := errors.New("error")
	email := r.FormValue("email")
	id := r.FormValue("id")

	// Get User from current token
	if email == "" && id == "" {
		user, err = a.getCurrentUser(r)
		if err != nil {
			httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
			return
		}
	}

	// Get User by email
	if email != "" {
		user, err = a.getUserByEmail(email)
		if err != nil {
			httputil.RespondWithJSON(w, http.StatusNotFound, err)
			return
		}
	}

	// Get User by ID
	if id != "" {
		user, err = a.getUserByID(id)
		if err != nil {
			httputil.RespondWithJSON(w, http.StatusNotFound, err)
			return
		}
	}

	// Get User Groups
	var groups []*gocloak.Group
	params := gocloak.GetGroupsParams{}
	ctx := context.Background()
	groups, err = a.client.GetUserGroups(ctx, a.token.AccessToken, "main", *user.ID, params)
	if err != nil {
		httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
		return
	}

	if len(groups) > 0 {
		for _, g := range groups {

			// Check if user in needed group
			if *g.ID == KenesOlami {
				httputil.RespondSuccess(w)
				return
			}
		}
	} else {
		httputil.RespondWithError(w, http.StatusNotFound, "No group found")
		return
	}

	httputil.RespondWithError(w, http.StatusNotFound, "failed")
}

func (a *App) getCurrentUser(r *http.Request) (*gocloak.User, error) {
	if rCtx, ok := middleware.ContextFromRequest(r); ok {
		if rCtx.IDClaims != nil {
			var user *gocloak.User
			ctx := context.Background()
			user, err := a.client.GetUserByID(ctx, a.token.AccessToken, "main", rCtx.IDClaims.Sub)
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
	var groups []*gocloak.Group
	params := gocloak.GetGroupsParams{}
	ctx := context.Background()
	groups, err = a.client.GetUserGroups(ctx, a.token.AccessToken, "main", *user.ID, params)
	if err != nil {
		return nil, err
	}

	// Make sure requested user in galaxy group
	if len(groups) > 0 {
		for _, g := range groups {
			if *g.ID == GalaxyUsers {
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
	attr := *cu.Attributes
	if _, ok := attr["request"]; ok {
		httputil.RespondWithError(w, http.StatusBadRequest, "Request Already Done")
		return
	}

	// Get requested user
	email := r.FormValue("email")
	ru, err := a.getRequestedUser(email)
	if err != nil {
		httputil.RespondWithJSON(w, http.StatusNotFound, err)
		return
	}

	// Set request attributes to requested user
	attr = *ru.Attributes
	if attr == nil {
		attr = map[string][]string{}
		attr["locale"] = []string{"en"}
	}
	if _, ok := attr["pending"]; ok {
		//val = append(val, *cu.Email)
		//ru.Attributes["pending"] = val
		httputil.RespondWithError(w, http.StatusBadRequest, "Pending Already Done")
		return
	} else {
		attr["pending"] = []string{*cu.Email}
	}

	ctx := context.Background()
	err = a.client.UpdateUser(ctx, a.token.AccessToken, "main", *ru)
	if err != nil {
		httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
		return
	}

	// Set request attributes to current user
	timestamp := int(time.Now().UnixNano() / int64(time.Millisecond))
	attr = *cu.Attributes
	if attr == nil {
		attr = map[string][]string{}
		attr["locale"] = []string{"en"}
	}
	attr["request"] = []string{email}
	attr["timestamp"] = []string{strconv.Itoa(timestamp)}
	err = a.client.UpdateUser(ctx, a.token.AccessToken, "main", *cu)
	if err != nil {
		httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
		return
	}

	httputil.RespondSuccess(w)
}

func (a *App) setPendingByMail(w http.ResponseWriter, r *http.Request) {

	// Get Pending User by Mail
	email := r.FormValue("email")
	cu, err := a.getUserByEmail(email)
	if err != nil {
		httputil.RespondWithJSON(w, http.StatusNotFound, err)
		return
	}

	// Get User Groups
	var groups []*gocloak.Group
	params := gocloak.GetGroupsParams{}
	ctx := context.Background()
	groups, err = a.client.GetUserGroups(ctx, a.token.AccessToken, "main", *cu.ID, params)
	if err != nil {
		httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
		return
	}

	if len(groups) > 0 {
		for _, g := range groups {

			// Make sure user in new group
			if *g.ID == NewUsers {

				// Move from new users to pending
				err = a.client.DeleteUserFromGroup(ctx, a.token.AccessToken, "main", *cu.ID, NewUsers)
				if err != nil {
					httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
					return
				}

				err = a.client.AddUserToGroup(ctx, a.token.AccessToken, "main", *cu.ID, GalaxyPending)
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

	httputil.RespondWithError(w, http.StatusNotFound, "Not in new group")
}

func (a *App) setPending(w http.ResponseWriter, r *http.Request) {

	// Get Current User
	cu, err := a.getCurrentUser(r)
	if err != nil {
		httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
		return
	}

	// Get User Groups
	var groups []*gocloak.Group
	params := gocloak.GetGroupsParams{}
	ctx := context.Background()
	groups, err = a.client.GetUserGroups(ctx, a.token.AccessToken, "main", *cu.ID, params)
	if err != nil {
		httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
		return
	}

	if len(groups) > 0 {
		for _, g := range groups {

			// Make sure user in new group
			if *g.ID == NewUsers {

				// Move from new users to pending
				err = a.client.DeleteUserFromGroup(ctx, a.token.AccessToken, "main", *cu.ID, NewUsers)
				if err != nil {
					httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
					return
				}

				err = a.client.AddUserToGroup(ctx, a.token.AccessToken, "main", *cu.ID, GalaxyPending)
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

	httputil.RespondWithError(w, http.StatusNotFound, "Not in new group")
}

func (a *App) verifyUser(w http.ResponseWriter, r *http.Request) {

	// Check role
	gxyUser := checkRole("gxy_user", r)
	if !gxyUser {
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
	var groups []*gocloak.Group
	params := gocloak.GetGroupsParams{}
	ctx := context.Background()
	groups, err = a.client.GetUserGroups(ctx, a.token.AccessToken, "main", *pu.ID, params)
	if err != nil {
		httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
		return
	}

	if len(groups) > 0 {
		for _, g := range groups {

			// Make sure pending user in pending group
			if *g.ID == "c46f3890-fa01-4933-968d-488ba5ca3153" {

				// Change pending User group
				err = a.client.DeleteUserFromGroup(ctx, a.token.AccessToken, "main", *pu.ID, "c46f3890-fa01-4933-968d-488ba5ca3153")
				if err != nil {
					httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
					return
				}

				err = a.client.AddUserToGroup(ctx, a.token.AccessToken, "main", *pu.ID, "04778f5d-31c1-4a2d-a395-7eac07ebc5b7")
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

	ctx := context.Background()
	// Set approved to current user attribute
	if action == "approve" {
		attr := *cu.Attributes
		if attr == nil {
			attr = map[string][]string{}
			attr["locale"] = []string{"en"}
		}
		if val, ok := attr["approved"]; ok {
			val = append(val, email)
			attr["approved"] = val
		} else {
			attr["approved"] = []string{email}

			// Add to verify group
			err = a.client.AddUserToGroup(ctx, a.token.AccessToken, "main", *cu.ID, "96a32920-5f34-4678-b8e3-ea26f4143558")
			if err != nil {
				return err
			}
		}

		// Remove request attribute from pending user
		attr = *pu.Attributes
		if _, ok := attr["request"]; ok {
			delete(attr, "request")
			delete(attr, "timestamp")
		}
	}

	// Remove pending attribute from current user
	attr := *cu.Attributes
	if attr == nil {
		attr = map[string][]string{}
		attr["locale"] = []string{"en"}
	}
	if val, ok := attr["pending"]; ok {
		if len(val) > 1 {
			for i, v := range val {
				if v == email {
					val = append(val[:i], val[i+1:]...)
					break
				}
			}
			attr["pending"] = val
		} else {
			delete(attr, "pending")
		}
	}
	err = a.client.UpdateUser(ctx, a.token.AccessToken, "main", *cu)
	if err != nil {
		return err
	}

	if action == "ignore" {
		// Add User to banned and ignored groups
		err = a.client.DeleteUserFromGroup(ctx, a.token.AccessToken, "main", *pu.ID, "c46f3890-fa01-4933-968d-488ba5ca3153")
		if err != nil {
			return err
		}

		err = a.client.AddUserToGroup(ctx, a.token.AccessToken, "main", *pu.ID, "c4569eaa-c67d-446e-b370-ad426a006c6b")
		if err != nil {
			return err
		}

		err = a.client.AddUserToGroup(ctx, a.token.AccessToken, "main", *pu.ID, "4111c55c-1931-4ca1-9f6f-127963d40dcd")
		if err != nil {
			return err
		}

		*pu.Enabled = false
	}

	err = a.client.UpdateUser(ctx, a.token.AccessToken, "main", *pu)
	if err != nil {
		return err
	}

	return nil
}

func (a *App) approveUserByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	ctx := context.Background()

	// Check role
	authAdmin := checkRole("auth_admin", r)
	if !authAdmin {
		e := errors.New("bad permission")
		httputil.NewUnauthorizedError(e).Abort(w, r)
		return
	}

	// Get User groups
	params := gocloak.GetGroupsParams{}
	groups, err := a.client.GetUserGroups(ctx, a.token.AccessToken, "main", id, params)
	if err != nil {
		httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
		return
	}

	for _, v := range groups {
		err := a.client.DeleteUserFromGroup(ctx, a.token.AccessToken, "main", id, *v.ID)
		if err != nil {
			httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
			return
		}
	}

	err = a.client.AddUserToGroup(ctx, a.token.AccessToken, "main", id, "04778f5d-31c1-4a2d-a395-7eac07ebc5b7")
	if err != nil {
		httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
		return
	}

	httputil.RespondSuccess(w)
}

func (a *App) kmediaGroup(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	// Get User by ID
	userId := r.FormValue("user_id")
	pu, err := a.client.GetUserByID(ctx, a.token.AccessToken, "main", userId)
	if err != nil {
		httputil.RespondWithJSON(w, http.StatusNotFound, err)
		return
	}

	// Add to kmedia group
	err = a.client.AddUserToGroup(ctx, a.token.AccessToken, "main", *pu.ID, KmediaUser)
	if err != nil {
		httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
		return
	}

	httputil.RespondSuccess(w)
	return
}

func (a *App) changeStatus(w http.ResponseWriter, r *http.Request) {

	groupId := r.FormValue("group_id")
	ctx := context.Background()

	//Allow only these groups in option
	if groupId == GalaxyGuests || groupId == GalaxyPending || groupId == BannedUsers || groupId == GalaxyUsers {

		// Get Pending User by ID
		userId := r.FormValue("user_id")
		pu, err := a.client.GetUserByID(ctx, a.token.AccessToken, "main", userId)
		if err != nil {
			httputil.RespondWithJSON(w, http.StatusNotFound, err)
			return
		}

		// Get User Groups
		var groups []*gocloak.Group
		params := gocloak.GetGroupsParams{}
		groups, err = a.client.GetUserGroups(ctx, a.token.AccessToken, "main", *pu.ID, params)
		if err != nil {
			httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
			return
		}

		// Remove only needed group
		if len(groups) > 0 {

			// Don't allow change user from banned and galaxy groups
			for _, v := range groups {
				if *v.ID == BannedUsers || *v.ID == GalaxyUsers {
					httputil.RespondWithError(w, http.StatusBadRequest, "Not allow changes")
					return
				}
			}

			// Remove user from pending or guests groups
			for _, v := range groups {
				if *v.ID == GalaxyPending || *v.ID == GalaxyGuests {
					err := a.client.DeleteUserFromGroup(ctx, a.token.AccessToken, "main", userId, *v.ID)
					if err != nil {
						httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
						return
					}
				}
			}

			// Add to requested group
			err = a.client.AddUserToGroup(ctx, a.token.AccessToken, "main", *pu.ID, groupId)
			if err != nil {
				httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
				return
			}

			// Send user notification
			if groupId == GalaxyUsers {
				go a.SendMessage(userId)
			}

			// Disable banned user
			if groupId == BannedUsers {
				*pu.Enabled = false
				err = a.client.UpdateUser(ctx, a.token.AccessToken, "main", *pu)
				if err != nil {
					httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
					return
				}
			}

			httputil.RespondSuccess(w)
			return

		} else {

			httputil.RespondWithError(w, http.StatusNotFound, "Not found any group")
			return

		}
	} else {

		httputil.RespondWithError(w, http.StatusBadRequest, "Not valid group id")
		return

	}
}

func (a *App) removeUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	ctx := context.Background()

	// Check role
	authRoot := checkRole("auth_root", r)
	if !authRoot {
		e := errors.New("bad permission")
		httputil.NewUnauthorizedError(e).Abort(w, r)
		return
	}

	err := a.client.DeleteUser(ctx, a.token.AccessToken, "main", id)
	if err != nil {
		httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
		return
	}

	httputil.RespondSuccess(w)
}

func (a *App) selfRemove(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	// Get Current User
	cu, err := a.getCurrentUser(r)
	if err != nil {
		httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
		return
	}

	err = a.client.DeleteUser(ctx, a.token.AccessToken, "main", *cu.ID)
	if err != nil {
		httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
		return
	}

	httputil.RespondSuccess(w)
}

func (a *App) cleanUsers(w http.ResponseWriter, r *http.Request) {
	// Check role
	authRoot := checkRole("auth_root", r)
	if !authRoot {
		e := errors.New("bad permission")
		httputil.NewUnauthorizedError(e).Abort(w, r)
		return
	}

	ctx := context.Background()
	max := 10000
	params := gocloak.GetGroupsParams{Max: &max}
	users, err := a.client.GetGroupMembers(ctx, a.token.AccessToken, "main", "c46f3890-fa01-4933-968d-488ba5ca3153", params)
	if err != nil {
		httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
		return
	}

	if len(users) > 0 {
		timenow := time.Now().UnixNano() / int64(time.Millisecond)
		curtime := int(time.Now().UnixNano() / int64(time.Millisecond))
		for _, u := range users {
			// Remove request attribute within 14 days
			attr := *u.Attributes
			if _, req := attr["request"]; req {
				if val, tim := attr["timestamp"]; tim {
					reqtime, _ := strconv.Atoi(val[0])
					if (curtime - reqtime) > 14*24*3600*1000 {
						delete(attr, "request")
						delete(attr, "timestamp")
						err = a.client.UpdateUser(ctx, a.token.AccessToken, "main", *u)
						if err != nil {
							httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
							return
						}
					}
				}
			}

			// Remove users with not verified mail within 7 days
			if *u.EmailVerified == false && (timenow-*u.CreatedTimestamp) > 7*24*3600*1000 {
				err = a.client.DeleteUser(ctx, a.token.AccessToken, "main", *u.ID)
				if err != nil {
					httputil.NewInternalError(pkgerr.WithStack(err)).Abort(w, r)
					return
				}
			}
		}
	}

	httputil.RespondSuccess(w)
}
