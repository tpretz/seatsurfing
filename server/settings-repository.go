package main

import (
	"strconv"
	"sync"
)

type SettingsRepository struct {
}

type OrgSetting struct {
	OrganizationID string
	Name           string
	Value          string
}

type SettingType int

const (
	SettingTypeInt      SettingType = 1
	SettingTypeBool     SettingType = 2
	SettingTypeString   SettingType = 3
	SettingTypeIntArray SettingType = 4
)

type SettingName struct {
	Name string
	Type SettingType
}

var (
	SettingInstallID                     SettingName = SettingName{Name: "install_id", Type: SettingTypeString}
	SettingDatabaseVersion               SettingName = SettingName{Name: "db_version", Type: SettingTypeInt}
	SettingAllowAnyUser                  SettingName = SettingName{Name: "allow_any_user", Type: SettingTypeBool}
	SettingConfluenceServerSharedSecret  SettingName = SettingName{Name: "confluence_server_shared_secret", Type: SettingTypeString}
	SettingConfluenceAnonymous           SettingName = SettingName{Name: "confluence_anonymous", Type: SettingTypeBool}
	SettingMaxBookingsPerUser            SettingName = SettingName{Name: "max_bookings_per_user", Type: SettingTypeInt}
	SettingMaxConcurrentBookingsPerUser  SettingName = SettingName{Name: "max_concurrent_bookings_per_user", Type: SettingTypeInt}
	SettingMaxDaysInAdvance              SettingName = SettingName{Name: "max_days_in_advance", Type: SettingTypeInt}
	SettingMaxBookingDurationHours       SettingName = SettingName{Name: "max_booking_duration_hours", Type: SettingTypeInt}
	SettingActiveSubscription            SettingName = SettingName{Name: "subscription_active", Type: SettingTypeBool}
	SettingDailyBasisBooking             SettingName = SettingName{Name: "daily_basis_booking", Type: SettingTypeBool}
	SettingShowNames                     SettingName = SettingName{Name: "show_names", Type: SettingTypeBool}
	SettingAllowBookingsNonExistingUsers SettingName = SettingName{Name: "allow_booking_nonexist_users", Type: SettingTypeBool}
	SettingSubscriptionMaxUsers          SettingName = SettingName{Name: "subscription_max_users", Type: SettingTypeInt}
	SettingDefaultTimezone               SettingName = SettingName{Name: "default_timezone", Type: SettingTypeString}
)

var settingsRepository *SettingsRepository
var settingsRepositoryOnce sync.Once

func GetSettingsRepository() *SettingsRepository {
	settingsRepositoryOnce.Do(func() {
		settingsRepository = &SettingsRepository{}
		_, err := GetDatabase().DB().Exec("CREATE TABLE IF NOT EXISTS settings (" +
			"organization_id uuid NOT NULL, " +
			"name VARCHAR NOT NULL, " +
			"value VARCHAR NOT NULL DEFAULT '', " +
			"PRIMARY KEY (organization_id, name))")
		if err != nil {
			panic(err)
		}
	})
	return settingsRepository
}

func (r *SettingsRepository) RunSchemaUpgrade(curVersion, targetVersion int) {
	// nothing yet
}

func (r *SettingsRepository) Set(organizationID string, name string, value string) error {
	_, err := GetDatabase().DB().Exec("INSERT INTO settings (organization_id, name, value) "+
		"VALUES ($1, $2, $3) "+
		"ON CONFLICT (organization_id, name) DO UPDATE SET value = $3",
		organizationID, name, value)
	return err
}

func (r *SettingsRepository) Get(organizationID string, name string) (string, error) {
	var res string
	err := GetDatabase().DB().QueryRow("SELECT value FROM settings "+
		"WHERE organization_id = $1 AND name = $2",
		organizationID, name).Scan(&res)
	if err != nil {
		return "", err
	}
	return res, nil
}

func (r *SettingsRepository) GetOrganizationIDsByValue(name, value string) ([]string, error) {
	var res []string
	rows, err := GetDatabase().DB().Query("SELECT organization_id FROM settings "+
		"WHERE name = $1 AND value = $2",
		name, value)
	if err != nil {
		return []string{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		err = rows.Scan(&id)
		if err != nil {
			return []string{}, err
		}
		res = append(res, id)
	}
	return res, nil
}

func (r *SettingsRepository) SetGlobal(name string, value string) error {
	return r.Set(r.getNullUUID(), name, value)
}

func (r *SettingsRepository) GetInt(organizationID string, name string) (int, error) {
	res, err := r.Get(organizationID, name)
	if err != nil {
		return 0, err
	}
	i, err := strconv.Atoi(res)
	return i, err
}

func (r *SettingsRepository) GetBool(organizationID string, name string) (bool, error) {
	res, err := r.Get(organizationID, name)
	if err != nil {
		return false, err
	}
	b := (res == "1")
	return b, err
}

func (r *SettingsRepository) GetGlobalString(name string) (string, error) {
	res, err := r.Get(r.getNullUUID(), name)
	if err != nil {
		return "", err
	}
	return res, nil
}

func (r *SettingsRepository) GetGlobalInt(name string) (int, error) {
	res, err := r.Get(r.getNullUUID(), name)
	if err != nil {
		return 0, err
	}
	i, err := strconv.Atoi(res)
	return i, err
}

func (r *SettingsRepository) GetGlobalBool(name string) (bool, error) {
	res, err := r.Get(r.getNullUUID(), name)
	if err != nil {
		return false, err
	}
	b := (res == "1")
	return b, err
}

func (r *SettingsRepository) GetAll(organizationID string) ([]*OrgSetting, error) {
	var result []*OrgSetting
	rows, err := GetDatabase().DB().Query("SELECT organization_id, name, value FROM settings "+
		"WHERE organization_id = $1 "+
		"ORDER BY name", organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		e := &OrgSetting{}
		err = rows.Scan(&e.OrganizationID, &e.Name, &e.Value)
		if err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, nil
}

func (r *SettingsRepository) InitDefaultSettingsForOrg(organizationID string) error {
	_, err := GetDatabase().DB().Exec("INSERT INTO settings (organization_id, name, value) "+
		"VALUES "+
		"($1, '"+SettingActiveSubscription.Name+"', '0'), "+
		"($1, '"+SettingSubscriptionMaxUsers.Name+"', '"+strconv.Itoa(GetConfig().OrgSignupMaxUsers)+"'), "+
		"($1, '"+SettingAllowAnyUser.Name+"', '1'), "+
		"($1, '"+SettingDailyBasisBooking.Name+"', '0'), "+
		"($1, '"+SettingShowNames.Name+"', '0'), "+
		"($1, '"+SettingAllowBookingsNonExistingUsers.Name+"', '0'), "+
		"($1, '"+SettingConfluenceServerSharedSecret.Name+"', ''), "+
		"($1, '"+SettingConfluenceAnonymous.Name+"', '0'), "+
		"($1, '"+SettingMaxBookingsPerUser.Name+"', '10'), "+
		"($1, '"+SettingMaxConcurrentBookingsPerUser.Name+"', '0'), "+
		"($1, '"+SettingMaxDaysInAdvance.Name+"', '14'), "+
		"($1, '"+SettingMaxBookingDurationHours.Name+"', '12'), "+
		"($1, '"+SettingDefaultTimezone.Name+"', 'Europe/Berlin') "+
		"ON CONFLICT (organization_id, name) DO NOTHING",
		organizationID)
	return err
}

func (r *SettingsRepository) InitDefaultSettings(orgIDs []string) error {
	for _, orgID := range orgIDs {
		if err := r.InitDefaultSettingsForOrg(orgID); err != nil {
			return err
		}
	}
	return nil
}

func (r *SettingsRepository) DeleteAll(organizationID string) error {
	_, err := GetDatabase().DB().Exec("DELETE FROM settings WHERE organization_id = $1", organizationID)
	return err
}

func (r *SettingsRepository) getNullUUID() string {
	return "00000000-0000-0000-0000-000000000000"
}
