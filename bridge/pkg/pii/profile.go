// Package pii provides PII management for ArmorClaw blind fill capability.
// Users store personal information profiles in an encrypted vault,
// and skills/agents request access via Human-in-the-Loop (HITL) consent.
package pii

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/armorclaw/bridge/pkg/securerandom"
)

// ProfileType defines the category of a user profile
type ProfileType string

const (
	ProfileTypePersonal   ProfileType = "personal"   // Personal identity info
	ProfileTypeBusiness   ProfileType = "business"   // Business/contact info
	ProfileTypePayment    ProfileType = "payment"    // Payment/billing info
	ProfileTypeMedical    ProfileType = "medical"    // Medical/health info
	ProfileTypeCustom     ProfileType = "custom"     // User-defined fields
)

// Standard field keys for profile data
const (
	FieldFullName    = "full_name"
	FieldFirstName   = "first_name"
	FieldLastName    = "last_name"
	FieldEmail       = "email"
	FieldPhone       = "phone"
	FieldDateOfBirth = "date_of_birth"
	FieldSSN         = "ssn"          // Social Security Number
	FieldAddress     = "address"
	FieldCity        = "city"
	FieldState       = "state"
	FieldPostalCode  = "postal_code"
	FieldCountry     = "country"
	FieldCompany     = "company"
	FieldJobTitle    = "job_title"
)

// ProfileData contains the actual PII values for a profile.
// All fields are optional - users fill only what they need.
type ProfileData struct {
	// Personal identity
	FullName    string `json:"full_name,omitempty"`
	FirstName   string `json:"first_name,omitempty"`
	LastName    string `json:"last_name,omitempty"`
	Email       string `json:"email,omitempty"`
	Phone       string `json:"phone,omitempty"`
	DateOfBirth string `json:"date_of_birth,omitempty"` // ISO 8601 format

	// Sensitive identifiers
	SSN string `json:"ssn,omitempty"` // Never logged, only accessed via consent

	// Address information
	Address    string `json:"address,omitempty"`
	City       string `json:"city,omitempty"`
	State      string `json:"state,omitempty"`
	PostalCode string `json:"postal_code,omitempty"`
	Country    string `json:"country,omitempty"`

	// Business information
	Company  string `json:"company,omitempty"`
	JobTitle string `json:"job_title,omitempty"`

	// Custom user-defined fields
	Custom map[string]string `json:"custom,omitempty"`
}

// ToMap converts ProfileData to a map for field lookup
func (pd *ProfileData) ToMap() map[string]string {
	result := make(map[string]string)

	if pd.FullName != "" {
		result[FieldFullName] = pd.FullName
	}
	if pd.FirstName != "" {
		result[FieldFirstName] = pd.FirstName
	}
	if pd.LastName != "" {
		result[FieldLastName] = pd.LastName
	}
	if pd.Email != "" {
		result[FieldEmail] = pd.Email
	}
	if pd.Phone != "" {
		result[FieldPhone] = pd.Phone
	}
	if pd.DateOfBirth != "" {
		result[FieldDateOfBirth] = pd.DateOfBirth
	}
	if pd.SSN != "" {
		result[FieldSSN] = pd.SSN
	}
	if pd.Address != "" {
		result[FieldAddress] = pd.Address
	}
	if pd.City != "" {
		result[FieldCity] = pd.City
	}
	if pd.State != "" {
		result[FieldState] = pd.State
	}
	if pd.PostalCode != "" {
		result[FieldPostalCode] = pd.PostalCode
	}
	if pd.Country != "" {
		result[FieldCountry] = pd.Country
	}
	if pd.Company != "" {
		result[FieldCompany] = pd.Company
	}
	if pd.JobTitle != "" {
		result[FieldJobTitle] = pd.JobTitle
	}

	// Add custom fields
	for k, v := range pd.Custom {
		result[k] = v
	}

	return result
}

// FieldDescriptor describes a single PII field for UI generation
type FieldDescriptor struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type"` // text, email, tel, date, etc.
	Required    bool   `json:"required"`
	Sensitive   bool   `json:"sensitive"` // If true, requires explicit consent
	Category    string `json:"category,omitempty"`
}

// ProfileFieldSchema describes available fields for a profile type
type ProfileFieldSchema struct {
	ProfileType ProfileType       `json:"profile_type"`
	Fields      []FieldDescriptor `json:"fields"`
}

// GetStandardFieldSchema returns the schema for a profile type
func GetStandardFieldSchema(profileType ProfileType) *ProfileFieldSchema {
	switch profileType {
	case ProfileTypePersonal:
		return &ProfileFieldSchema{
			ProfileType: ProfileTypePersonal,
			Fields: []FieldDescriptor{
				{Key: FieldFullName, Label: "Full Name", Type: "text", Category: "identity"},
				{Key: FieldFirstName, Label: "First Name", Type: "text", Category: "identity"},
				{Key: FieldLastName, Label: "Last Name", Type: "text", Category: "identity"},
				{Key: FieldEmail, Label: "Email Address", Type: "email", Category: "contact"},
				{Key: FieldPhone, Label: "Phone Number", Type: "tel", Category: "contact"},
				{Key: FieldDateOfBirth, Label: "Date of Birth", Type: "date", Sensitive: true, Category: "identity"},
				{Key: FieldSSN, Label: "Social Security Number", Type: "text", Sensitive: true, Category: "identity"},
				{Key: FieldAddress, Label: "Street Address", Type: "text", Category: "location"},
				{Key: FieldCity, Label: "City", Type: "text", Category: "location"},
				{Key: FieldState, Label: "State/Province", Type: "text", Category: "location"},
				{Key: FieldPostalCode, Label: "Postal Code", Type: "text", Category: "location"},
				{Key: FieldCountry, Label: "Country", Type: "text", Category: "location"},
			},
		}
	case ProfileTypeBusiness:
		return &ProfileFieldSchema{
			ProfileType: ProfileTypeBusiness,
			Fields: []FieldDescriptor{
				{Key: FieldFullName, Label: "Full Name", Type: "text", Category: "identity"},
				{Key: FieldEmail, Label: "Work Email", Type: "email", Category: "contact"},
				{Key: FieldPhone, Label: "Work Phone", Type: "tel", Category: "contact"},
				{Key: FieldCompany, Label: "Company Name", Type: "text", Category: "business"},
				{Key: FieldJobTitle, Label: "Job Title", Type: "text", Category: "business"},
				{Key: FieldAddress, Label: "Business Address", Type: "text", Category: "location"},
				{Key: FieldCity, Label: "City", Type: "text", Category: "location"},
				{Key: FieldState, Label: "State/Province", Type: "text", Category: "location"},
				{Key: FieldPostalCode, Label: "Postal Code", Type: "text", Category: "location"},
				{Key: FieldCountry, Label: "Country", Type: "text", Category: "location"},
			},
		}
	case ProfileTypeCustom:
		return &ProfileFieldSchema{
			ProfileType: ProfileTypeCustom,
			Fields:      []FieldDescriptor{}, // Custom profiles define their own fields
		}
	default:
		return &ProfileFieldSchema{
			ProfileType: profileType,
			Fields:      []FieldDescriptor{},
		}
	}
}

// UserProfile represents a stored PII profile in the encrypted vault
type UserProfile struct {
	ID           string            `json:"id"`
	ProfileName  string            `json:"profile_name"`
	ProfileType  ProfileType       `json:"profile_type"`
	Data         ProfileData       `json:"data"`          // Decrypted PII values
	FieldSchema ProfileFieldSchema `json:"field_schema"`  // Describes available fields
	CreatedAt    int64             `json:"created_at"`
	UpdatedAt    int64             `json:"updated_at"`
	LastAccessed int64             `json:"last_accessed,omitempty"`
	IsDefault    bool              `json:"is_default"`
}

// ProfileInfo is the public view of a profile (no PII values)
type ProfileInfo struct {
	ID           string      `json:"id"`
	ProfileName  string      `json:"profile_name"`
	ProfileType  ProfileType `json:"profile_type"`
	CreatedAt    int64       `json:"created_at"`
	UpdatedAt    int64       `json:"updated_at"`
	LastAccessed int64       `json:"last_accessed,omitempty"`
	IsDefault    bool        `json:"is_default"`
	FieldCount   int         `json:"field_count"` // Number of non-empty fields
}

// NewUserProfile creates a new user profile with generated ID
func NewUserProfile(profileName string, profileType ProfileType) *UserProfile {
	now := time.Now().Unix()
	return &UserProfile{
		ID:          "profile_" + securerandom.MustID(16),
		ProfileName: profileName,
		ProfileType: profileType,
		Data:        ProfileData{Custom: make(map[string]string)},
		FieldSchema: *GetStandardFieldSchema(profileType),
		CreatedAt:   now,
		UpdatedAt:   now,
		IsDefault:   false,
	}
}

// ToInfo converts a UserProfile to ProfileInfo (public view without PII)
func (p *UserProfile) ToInfo() *ProfileInfo {
	return &ProfileInfo{
		ID:           p.ID,
		ProfileName:  p.ProfileName,
		ProfileType:  p.ProfileType,
		CreatedAt:    p.CreatedAt,
		UpdatedAt:    p.UpdatedAt,
		LastAccessed: p.LastAccessed,
		IsDefault:    p.IsDefault,
		FieldCount:   p.countNonEmptyFields(),
	}
}

// countNonEmptyFields counts the number of non-empty fields
func (p *UserProfile) countNonEmptyFields() int {
	count := 0
	data := p.Data

	if data.FullName != "" {
		count++
	}
	if data.FirstName != "" {
		count++
	}
	if data.LastName != "" {
		count++
	}
	if data.Email != "" {
		count++
	}
	if data.Phone != "" {
		count++
	}
	if data.DateOfBirth != "" {
		count++
	}
	if data.SSN != "" {
		count++
	}
	if data.Address != "" {
		count++
	}
	if data.City != "" {
		count++
	}
	if data.State != "" {
		count++
	}
	if data.PostalCode != "" {
		count++
	}
	if data.Country != "" {
		count++
	}
	if data.Company != "" {
		count++
	}
	if data.JobTitle != "" {
		count++
	}
	count += len(data.Custom)

	return count
}

// GetField retrieves a field value by key
func (p *UserProfile) GetField(key string) (string, bool) {
	dataMap := p.Data.ToMap()
	val, exists := dataMap[key]
	return val, exists
}

// SetField sets a field value by key
func (p *UserProfile) SetField(key, value string) {
	switch key {
	case FieldFullName:
		p.Data.FullName = value
	case FieldFirstName:
		p.Data.FirstName = value
	case FieldLastName:
		p.Data.LastName = value
	case FieldEmail:
		p.Data.Email = value
	case FieldPhone:
		p.Data.Phone = value
	case FieldDateOfBirth:
		p.Data.DateOfBirth = value
	case FieldSSN:
		p.Data.SSN = value
	case FieldAddress:
		p.Data.Address = value
	case FieldCity:
		p.Data.City = value
	case FieldState:
		p.Data.State = value
	case FieldPostalCode:
		p.Data.PostalCode = value
	case FieldCountry:
		p.Data.Country = value
	case FieldCompany:
		p.Data.Company = value
	case FieldJobTitle:
		p.Data.JobTitle = value
	default:
		// Custom field
		if p.Data.Custom == nil {
			p.Data.Custom = make(map[string]string)
		}
		p.Data.Custom[key] = value
	}
	p.UpdatedAt = time.Now().Unix()
}

// Validate checks if the profile data is valid
func (p *UserProfile) Validate() error {
	if p.ProfileName == "" {
		return errors.New("profile name is required")
	}
	if p.ProfileType == "" {
		return errors.New("profile type is required")
	}
	return nil
}

// MarshalData serializes profile data to JSON for encryption
func (p *UserProfile) MarshalData() ([]byte, error) {
	return json.Marshal(p.Data)
}

// UnmarshalData deserializes profile data from decrypted JSON
func (p *UserProfile) UnmarshalData(data []byte) error {
	return json.Unmarshal(data, &p.Data)
}

// MarshalSchema serializes field schema to JSON for storage
func (p *UserProfile) MarshalSchema() (string, error) {
	data, err := json.Marshal(p.FieldSchema)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// UnmarshalSchema deserializes field schema from stored JSON
func (p *UserProfile) UnmarshalSchema(data string) error {
	return json.Unmarshal([]byte(data), &p.FieldSchema)
}

// GetSensitiveFields returns a list of field keys that are marked as sensitive
func (p *UserProfile) GetSensitiveFields() []string {
	var sensitive []string
	for _, field := range p.FieldSchema.Fields {
		if field.Sensitive {
			sensitive = append(sensitive, field.Key)
		}
	}
	return sensitive
}

// IsFieldSensitive checks if a field is marked as sensitive
func (p *UserProfile) IsFieldSensitive(key string) bool {
	for _, field := range p.FieldSchema.Fields {
		if field.Key == key && field.Sensitive {
			return true
		}
	}
	// Custom fields are not sensitive by default
	return false
}
