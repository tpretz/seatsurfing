package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

type GroupRouter struct {
}

type CreateGroupRequest struct {
	Name           string `json:"name" validate:"required"`
	Type           int    `json:"type"`
	Description    string `json:"description"`
	OrganizationID string `json:"organizationId"`
}

type GetGroupResponse struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Type           int    `json:"type"`
	Description    string `json:"description"`
	OrganizationID string `json:"organizationId"`
}

type GetGroupCountResponse struct {
	Count int `json:"count"`
}

func (router *GroupRouter) setupRoutes(s *mux.Router) {
	s.HandleFunc("/count", router.getCount).Methods("GET")
	s.HandleFunc("/me", router.getSelf).Methods("GET")
	s.HandleFunc("/{id}", router.getOne).Methods("GET")
	//	s.HandleFunc("/{id}", router.update).Methods("PUT")
	s.HandleFunc("/{id}", router.delete).Methods("DELETE")
	s.HandleFunc("/", router.create).Methods("POST")
	s.HandleFunc("/", router.getAll).Methods("GET")
}

func (router *GroupRouter) getCount(w http.ResponseWriter, r *http.Request) {
	user := GetRequestUser(r)
	if !CanAdminOrg(user, user.OrganizationID) {
		SendForbidden(w)
		return
	}
	num, _ := GetGroupRepository().GetCount(user.OrganizationID)
	m := &GetGroupCountResponse{
		Count: num,
	}
	SendJSON(w, m)
}

func (router *GroupRouter) getSelf(w http.ResponseWriter, r *http.Request) {
	e := GetRequestUser(r)
	if e == nil {
		SendNotFound(w)
		return
	}
	groups, err := e.Groups()
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}

	var groupNames []string

	for _, group := range groups {
		groupNames = append(groupNames, group.Name)
	}

	SendJSON(w, groupNames)
}

// need to include members
func (router *GroupRouter) getOne(w http.ResponseWriter, r *http.Request) {
	user := GetRequestUser(r)
	if !CanAdminOrg(user, user.OrganizationID) {
		SendForbidden(w)
		return
	}
	vars := mux.Vars(r)
	e, err := GetGroupRepository().GetOneWithMembers(vars["id"])
	if err != nil {
		log.Println(err)
		SendNotFound(w)
		return
	}
	if e.OrganizationID != user.OrganizationID {
		SendForbidden(w)
		return
	}

	res := router.copyToRestModel(e, true)
	SendJSON(w, res)
}

func (router *GroupRouter) getAll(w http.ResponseWriter, r *http.Request) {
	user := GetRequestUser(r)
	if !CanAdminOrg(user, user.OrganizationID) {
		SendForbidden(w)
		return
	}
	list, err := GetGroupRepository().GetAll(user.OrganizationID, 1000, 0)
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	res := []*GetGroupResponse{}
	for _, e := range list {
		m := router.copyToRestModel(e, true)
		res = append(res, m)
	}
	SendJSON(w, res)
}

// func (router *GroupRouter) update(w http.ResponseWriter, r *http.Request) {
// 	var m CreateGroupRequest
// 	if UnmarshalValidateBody(r, &m) != nil {
// 		SendBadRequest(w)
// 		return
// 	}
// 	vars := mux.Vars(r)
// 	e, err := GetGroupRepository().GetOne(vars["id"])
// 	if err != nil {
// 		SendBadRequest(w)
// 		return
// 	}
// 	user := GetRequestUser(r)
// 	if !CanAdminOrg(user, e.OrganizationID) {
// 		SendForbidden(w)
// 		return
// 	}
// 	eNew := router.copyFromRestModel(&m)
// 	eNew.ID = e.ID
// 	if eNew.Role > user.Role {
// 		eNew.Role = e.Role
// 	}
// 	eNew.OrganizationID = e.OrganizationID
// 	eNew.HashedPassword = e.HashedPassword
// 	org, err := GetOrganizationRepository().GetOne(e.OrganizationID)
// 	if err != nil {
// 		log.Println(err)
// 		SendInternalServerError(w)
// 		return
// 	}
// 	if !GetOrganizationRepository().isValidEmailForOrg(user.Email, org) {
// 		SendBadRequest(w)
// 		return
// 	}
// 	if err := GetUserRepository().Update(eNew); err != nil {
// 		log.Println(err)
// 		SendInternalServerError(w)
// 		return
// 	}
// 	SendUpdated(w)
// }

func (router *GroupRouter) delete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	e, err := GetGroupRepository().GetOne(vars["id"])
	if err != nil {
		SendNotFound(w)
		return
	}
	user := GetRequestUser(r)
	if !CanAdminOrg(user, e.OrganizationID) {
		SendForbidden(w)
		return
	}
	if err := e.Delete(); err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	SendUpdated(w)
}

func (router *GroupRouter) create(w http.ResponseWriter, r *http.Request) {
	user := GetRequestUser(r)
	if !CanAdminOrg(user, user.OrganizationID) {
		SendForbidden(w)
		return
	}
	var m CreateGroupRequest
	if UnmarshalValidateBody(r, &m) != nil {
		SendBadRequest(w)
		return
	}
	if m.OrganizationID != "" && m.OrganizationID != user.OrganizationID && !GetUserRepository().isSuperAdmin(user) {
		SendForbidden(w)
		return
	}
	e := router.copyFromRestModel(&m)
	e.Type = GroupTypeLocal
	if e.OrganizationID == "" || !GetUserRepository().isSuperAdmin(user) {
		e.OrganizationID = user.OrganizationID
	}
	org, err := GetOrganizationRepository().GetOne(e.OrganizationID)
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	if !GetGroupRepository().canCreateGroup(org) {
		SendPaymentRequired(w)
		return
	}
	if err := GetGroupRepository().Create(e); err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	SendCreated(w, e.ID)
}

func (router *GroupRouter) copyFromRestModel(m *CreateGroupRequest) *Group {
	e := &Group{}
	e.Name = m.Name
	e.Type = GroupType(m.Type)
	e.OrganizationID = m.OrganizationID
	e.Description = NullString(m.Description)
	return e
}

func (router *GroupRouter) copyToRestModel(e *Group, admin bool) *GetGroupResponse {
	m := &GetGroupResponse{}
	m.ID = e.ID
	m.OrganizationID = e.OrganizationID
	m.Name = e.Name
	m.Description = string(e.Description)
	m.Type = int(e.Type)
	return m
}
