package secretary

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/armorclaw/bridge/pkg/keystore"
	"github.com/armorclaw/bridge/pkg/logger"
	"github.com/google/uuid"
)

//=============================================================================
// Rolodex Service
//=============================================================================

// RolodexService manages contact storage with encryption for sensitive fields
type RolodexService struct {
	store Store
	ks    *keystore.Keystore
	log   *logger.Logger
}

// RolodexConfig holds configuration for the Rolodex service
type RolodexConfig struct {
	Store    Store
	Keystore *keystore.Keystore
	Logger   *logger.Logger
}

// NewRolodexService creates a new Rolodex service instance
func NewRolodexService(cfg RolodexConfig) (*RolodexService, error) {
	if cfg.Store == nil {
		return nil, fmt.Errorf("store is required")
	}
	if cfg.Keystore == nil {
		return nil, fmt.Errorf("keystore is required")
	}

	log := cfg.Logger
	if log == nil {
		log = logger.Global().WithComponent("rolodex")
	}

	return &RolodexService{
		store: cfg.Store,
		ks:    cfg.Keystore,
		log:   log,
	}, nil
}

// ContactData contains the sensitive contact fields that will be encrypted
type ContactData struct {
	Phone   string `json:"phone,omitempty"`
	Email   string `json:"email,omitempty"`
	Address string `json:"address,omitempty"`
	Notes   string `json:"notes,omitempty"`
}

// CreateRequest is the input for creating a contact
type CreateRequest struct {
	Name         string
	Company      string
	Relationship string
	Phone        string
	Email        string
	Address      string
	Notes        string
	CreatedBy    string
}

// CreateContact creates a new contact with encrypted sensitive fields
func (r *RolodexService) CreateContact(ctx context.Context, req *CreateRequest) (*Contact, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if req.CreatedBy == "" {
		return nil, fmt.Errorf("created_by is required")
	}

	contactID := uuid.New().String()

	// Prepare sensitive data for encryption
	sensitiveData := ContactData{
		Phone:   req.Phone,
		Email:   req.Email,
		Address: req.Address,
		Notes:   req.Notes,
	}

	// Serialize to JSON
	dataJSON, err := json.Marshal(sensitiveData)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize contact data: %w", err)
	}

	// Encrypt the sensitive data
	encrypted, nonce, err := r.encryptData(dataJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt contact data: %w", err)
	}

	now := time.Now()

	contact := &Contact{
		ID:             contactID,
		Name:           req.Name,
		Company:        req.Company,
		Relationship:   req.Relationship,
		EncryptedData:  encrypted,
		EncryptedNonce: nonce,
		CreatedBy:      req.CreatedBy,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := r.store.CreateContact(ctx, contact); err != nil {
		return nil, fmt.Errorf("failed to store contact: %w", err)
	}

	r.log.Info("contact_created", "contact_id", contactID, "name", req.Name, "created_by", req.CreatedBy)

	return contact, nil
}

// GetContact retrieves a contact and decrypts sensitive fields
func (r *RolodexService) GetContact(ctx context.Context, id string) (*Contact, error) {
	contact, err := r.store.GetContact(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := r.decryptContact(contact); err != nil {
		return nil, fmt.Errorf("failed to decrypt contact: %w", err)
	}

	return contact, nil
}

// ListContacts lists contacts with optional filtering
func (r *RolodexService) ListContacts(ctx context.Context, filter ContactFilter) ([]Contact, error) {
	contacts, err := r.store.ListContacts(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Decrypt sensitive fields for each contact
	for i := range contacts {
		if err := r.decryptContact(&contacts[i]); err != nil {
			r.log.Warn("contact_decrypt_failed", "contact_id", contacts[i].ID, "error", err)
			contacts[i].Phone = ""
			contacts[i].Email = ""
			contacts[i].Address = ""
			contacts[i].Notes = ""
		}
	}

	return contacts, nil
}

// UpdateRequest is the input for updating a contact
type UpdateRequest struct {
	ID           string
	Name         string
	Company      string
	Relationship string
	Phone        string
	Email        string
	Address      string
	Notes        string
	UpdatedBy    string
}

// UpdateContact updates an existing contact with encrypted sensitive fields
func (r *RolodexService) UpdateContact(ctx context.Context, req *UpdateRequest) (*Contact, error) {
	if req.ID == "" {
		return nil, fmt.Errorf("id is required")
	}

	existing, err := r.store.GetContact(ctx, req.ID)
	if err != nil {
		return nil, fmt.Errorf("contact not found: %w", err)
	}

	// Prepare sensitive data for encryption (use provided values or existing ones)
	sensitiveData := ContactData{
		Phone:   req.Phone,
		Email:   req.Email,
		Address: req.Address,
		Notes:   req.Notes,
	}

	if req.Phone == "" && req.Email == "" && req.Address == "" && req.Notes == "" {
		sensitiveData.Phone = existing.Phone
		sensitiveData.Email = existing.Email
		sensitiveData.Address = existing.Address
		sensitiveData.Notes = existing.Notes
	}

	// Serialize to JSON
	dataJSON, err := json.Marshal(sensitiveData)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize contact data: %w", err)
	}

	// Encrypt the sensitive data
	encrypted, nonce, err := r.encryptData(dataJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt contact data: %w", err)
	}

	// Update the contact with encrypted data
	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Company != "" {
		existing.Company = req.Company
	}
	if req.Relationship != "" {
		existing.Relationship = req.Relationship
	}
	existing.EncryptedData = encrypted
	existing.EncryptedNonce = nonce
	existing.UpdatedAt = time.Now()

	if err := r.store.UpdateContact(ctx, existing); err != nil {
		return nil, fmt.Errorf("failed to update contact: %w", err)
	}

	r.log.Info("contact_updated", "contact_id", req.ID, "updated_by", req.UpdatedBy)

	// Decrypt and return
	if err := r.decryptContact(existing); err != nil {
		return nil, fmt.Errorf("failed to decrypt contact: %w", err)
	}

	return existing, nil
}

// DeleteContact removes a contact
func (r *RolodexService) DeleteContact(ctx context.Context, id string) error {
	if err := r.store.DeleteContact(ctx, id); err != nil {
		return err
	}

	r.log.Info("contact_deleted", "contact_id", id)

	return nil
}

// SearchContacts searches for contacts by name, company, or relationship
func (r *RolodexService) SearchContacts(ctx context.Context, query string) ([]Contact, error) {
	filter := ContactFilter{
		Name:    query,
		Company: query,
	}

	contacts, err := r.store.ListContacts(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Decrypt sensitive fields for each contact
	for i := range contacts {
		if err := r.decryptContact(&contacts[i]); err != nil {
			r.log.Warn("contact_decrypt_failed", "contact_id", contacts[i].ID, "error", err)
			contacts[i].Phone = ""
			contacts[i].Email = ""
			contacts[i].Address = ""
			contacts[i].Notes = ""
		}
	}

	return contacts, nil
}

// encryptData encrypts data using the keystore
func (r *RolodexService) encryptData(data []byte) (encrypted, nonce []byte, err error) {
	return r.ks.Encrypt(data)
}

// decryptContact decrypts sensitive fields in a contact
func (r *RolodexService) decryptContact(contact *Contact) error {
	if contact.EncryptedData == nil || len(contact.EncryptedData) == 0 {
		return nil
	}

	decrypted, err := r.ks.Decrypt(contact.EncryptedData, contact.EncryptedNonce)
	if err != nil {
		return fmt.Errorf("decryption failed: %w", err)
	}

	var sensitiveData ContactData
	if err := json.Unmarshal(decrypted, &sensitiveData); err != nil {
		return fmt.Errorf("failed to deserialize contact data: %w", err)
	}

	contact.Phone = sensitiveData.Phone
	contact.Email = sensitiveData.Email
	contact.Address = sensitiveData.Address
	contact.Notes = sensitiveData.Notes

	return nil
}
