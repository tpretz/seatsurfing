package main

import (
	"strings"
	"sync"
)

type GroupRepository struct {
}

type GroupType int

const (
	GroupTypeLocal  GroupType = 0
	GroupTypeRemote GroupType = 10
)

type Group struct {
	ID             string
	OrganizationID string
	Name           string
	Description    NullString
	Type           GroupType
}

var groupRepository *GroupRepository
var groupRepositoryOnce sync.Once

func GetGroupRepository() *GroupRepository {
	groupRepositoryOnce.Do(func() {
		groupRepository = &GroupRepository{}
		_, err := GetDatabase().DB().Exec("CREATE TABLE IF NOT EXISTS groups (" + // will need a join table too
			"id uuid DEFAULT uuid_generate_v4(), " +
			"organization_id uuid NOT NULL, " +
			"name VARCHAR NOT NULL, " +
			"description VARCHAR, " +
			"type INT, " +
			"PRIMARY KEY (id))")
		if err != nil {
			panic(err)
		}
		_, err = GetDatabase().DB().Exec("CREATE UNIQUE INDEX IF NOT EXISTS group_name ON groups(name)")
		if err != nil {
			panic(err)
		}
	})
	return groupRepository
}

func (r *GroupRepository) RunSchemaUpgrade(curVersion, targetVersion int) {
}

func (r *GroupRepository) Create(e *Group) error {
	var id string
	err := GetDatabase().DB().QueryRow("INSERT INTO groups "+
		"(organization_id, name, description, type) "+
		"VALUES ($1, $2, $3, $4) "+
		"RETURNING id",
		e.OrganizationID, strings.ToLower(e.Name), CheckNullString(e.Description), e.Type).Scan(&id)
	if err != nil {
		return err
	}
	e.ID = id
	GetUserPreferencesRepository().InitDefaultSettingsForUser(e.ID)
	return nil
}

func (r *GroupRepository) GetOne(id string) (*Group, error) {
	e := &Group{}
	err := GetDatabase().DB().QueryRow("SELECT id, organization_id, name, description, type "+
		"FROM groups "+
		"WHERE id = $1",
		id).Scan(&e.ID, &e.OrganizationID, &e.Name, &e.Description, &e.Type)
	if err != nil {
		return nil, err
	}
	return e, nil
}

func (r *GroupRepository) GetByName(name string) (*Group, error) {
	e := &Group{}
	err := GetDatabase().DB().QueryRow("SELECT id, organization_id, name, description, type "+
		"FROM groups "+
		"WHERE LOWER(name) = $1",
		strings.ToLower(name)).Scan(&e.ID, &e.OrganizationID, &e.Name, &e.Description, &e.Type)
	if err != nil {
		return nil, err
	}
	return e, nil
}

func (r *GroupRepository) GetAll(organizationID string, maxResults int, offset int) ([]*Group, error) {
	var result []*Group
	rows, err := GetDatabase().DB().Query("SELECT id, organization_id, name, description, type "+
		"FROM groups "+
		"WHERE organization_id = $1 "+
		"ORDER BY name "+
		"LIMIT $2 OFFSET $3", organizationID, maxResults, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		e := &Group{}
		err = rows.Scan(&e.ID, &e.OrganizationID, &e.Name, &e.Description, &e.Type)
		if err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, nil
}

func (r *GroupRepository) GetAllIDs() ([]string, error) {
	var result []string
	rows, err := GetDatabase().DB().Query("SELECT id " +
		"FROM groups")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var ID string
		err = rows.Scan(&ID)
		if err != nil {
			return nil, err
		}
		result = append(result, ID)
	}
	return result, nil
}

func (r *GroupRepository) Update(e *Group) error {
	_, err := GetDatabase().DB().Exec("UPDATE groups SET "+
		"organization_id = $1, "+
		"name = $2, "+
		"description = $3, "+
		"type = $4, "+
		"WHERE id = $5",
		e.OrganizationID, strings.ToLower(e.Name), CheckNullString(e.Description), e.Type, e.ID)
	return err
}

func (r *GroupRepository) Delete(e *Group) error {
	// if _, err := GetDatabase().DB().Exec("DELETE FROM bookings WHERE "+ // need to delete from join table if no cascade
	// 	"bookings.user_id = $1", e.ID); err != nil {
	// 	return err
	// }
	_, err := GetDatabase().DB().Exec("DELETE FROM groups WHERE id = $1", e.ID)
	return err
}

func (r *GroupRepository) DeleteAll(organizationID string) error {
	// need to clean up any join tables too
	_, err := GetDatabase().DB().Exec("DELETE FROM groups WHERE organization_id = $1", organizationID)
	return err
}

func (r *GroupRepository) GetCount(organizationID string) (int, error) {
	var res int
	err := GetDatabase().DB().QueryRow("SELECT COUNT(id) "+
		"FROM groups "+
		"WHERE organization_id = $1",
		organizationID).Scan(&res)
	return res, err
}

func (r *GroupRepository) canCreateGroup(org *Organization) bool {
	maxGroups, _ := GetSettingsRepository().GetInt(org.ID, SettingSubscriptionMaxGroups.Name)
	curGroups, _ := GetGroupRepository().GetCount(org.ID)
	return curGroups < maxGroups
}

// add functions to object for add member, remove member, list members, etc.
func (g *Group) AddMember(user *User) error {
	_, err := GetDatabase().DB().Exec("INSERT INTO group_members "+
		"(group_id, user_id) "+
		"VALUES ($1, $2)",
		g.ID, user.ID)
	return err
}

func (g *Group) RemoveMember(user *User) error {
	_, err := GetDatabase().DB().Exec("DELETE FROM group_members "+
		"WHERE group_id = $1 AND user_id = $2",
		g.ID, user.ID)
	return err
}

func (g *Group) GetMembers() ([]*User, error) {
	var result []*User
	// only return fields in the User object
	rows, err := GetDatabase().DB().Query("SELECT u.id, u.organization_id, u.email, u.password, u.first_name, u.last_name, u.role, u.last_login, u.created, u.updated, u.deleted "+
		"FROM users u "+
		"INNER JOIN group_members gm ON gm.user_id = u.id "+
		"WHERE gm.group_id = $1 "+
		"ORDER BY u.last_name, u.first_name", g.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		u := &User{}
		err = rows.Scan(&u.ID, &u.OrganizationID, &u.Email, &u.Password, &u.FirstName, &u.LastName, &u.Role, &u.LastLogin, &u.Created, &u.Updated, &u.Deleted)
		if err != nil {
			return nil, err
		}
		result = append(result, u)
	}
	return result, nil
}