package main

import (
	"fmt"
	"testing"
)

func TestGroupsCRUD(t *testing.T) {
	clearTestDB()

	// Create
	group := &Group{
		Name:           "test_group",
		Description:    "this is a test group",
		OrganizationID: "73980078-f4d7-40ff-9211-a7bcbf8d1981",
	}
	GetGroupRepository().Create(group)
	checkStringNotEmpty(t, group.ID)

	// Read
	group2, err := GetGroupRepository().GetOne(group.ID)
	if err != nil {
		t.Fatalf("Expected non-nil group")
	}
	checkTestString(t, group.ID, group2.ID)
	checkTestString(t, "73980078-f4d7-40ff-9211-a7bcbf8d1981", group2.OrganizationID)

	// Update
	group2.OrganizationID = "61bf23af-0310-4d2b-b401-21c31d60c2c4"
	err = group2.Update()
	if err != nil {
		panic(err)
	}

	// Read
	group3, err := GetGroupRepository().GetOne(group.ID)
	if err != nil {
		t.Fatalf("Expected non-nil group")
	}
	checkTestString(t, group.ID, group3.ID)
	checkTestString(t, "61bf23af-0310-4d2b-b401-21c31d60c2c4", group3.OrganizationID)

	// Delete
	group.Delete()
	_, err = GetGroupRepository().GetOne(group.ID)
	if err == nil {
		t.Fatalf("Expected nil group")
	}
}

func TestGroupsCount(t *testing.T) {
	clearTestDB()
	org := createTestOrg("test.com")
	createTestGroupInOrg(org)
	createTestGroupInOrg(org)

	res, err := GetGroupRepository().GetCount(org.ID)
	if err != nil {
		t.Fatal(err)
	}
	checkTestInt(t, 2, res)
}

func TestGroupsMembers(t *testing.T) {
	clearTestDB()
	org := createTestOrg("test.com")
	grp := createTestGroupInOrg(org)

	usra := createTestUserInOrg(org);
	usrb := createTestUserInOrg(org);

	res, err := GetGroupRepository().GetOne(grp.ID)
	if err != nil {
		t.Fatal(err)
	}

	err = res.AddMember(usra)
	if err != nil {
		t.Fatal(err)
	}
	err = res.AddMember(usrb)
	if err != nil {
		t.Fatal(err)
	}

	members, err := res.Members()
	if err != nil {
		t.Fatal(err)
	}
	checkTestInt(t, 2, len(members))

	//fmt.Printf("members: %s\n", members[0].Email)
}
