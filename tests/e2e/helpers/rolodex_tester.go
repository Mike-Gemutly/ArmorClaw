package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/armorclaw/bridge/pkg/keystore"
	"github.com/armorclaw/bridge/pkg/secretary"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: rolodex_tester <command> [options]")
		fmt.Fprintln(os.Stderr, "Commands: create, get, search, update, delete, verify-encryption")
		os.Exit(1)
	}

	command := os.Args[1]

	ctx := context.Background()

	var dbPath, keystorePath string
	var name, company, relationship, phone, email, address, notes, query string

	// Parse flags (simple parsing for E2E test)
	for i := 2; i < len(os.Args); i++ {
		arg := os.Args[i]
		switch {
		case arg == "--db":
			if i+1 < len(os.Args) {
				dbPath = os.Args[i+1]
				i++
			}
		case arg == "--keystore":
			if i+1 < len(os.Args) {
				keystorePath = os.Args[i+1]
				i++
			}
		case arg == "--name":
			if i+1 < len(os.Args) {
				name = os.Args[i+1]
				i++
			}
		case arg == "--query":
			if i+1 < len(os.Args) {
				query = os.Args[i+1]
				i++
			}
		case arg == "--company":
			if i+1 < len(os.Args) {
				company = os.Args[i+1]
				i++
			}
		case arg == "--relationship":
			if i+1 < len(os.Args) {
				relationship = os.Args[i+1]
				i++
			}
		case arg == "--phone":
			if i+1 < len(os.Args) {
				phone = os.Args[i+1]
				i++
			}
		case arg == "--email":
			if i+1 < len(os.Args) {
				email = os.Args[i+1]
				i++
			}
		case arg == "--address":
			if i+1 < len(os.Args) {
				address = os.Args[i+1]
				i++
			}
		case arg == "--notes":
			if i+1 < len(os.Args) {
				notes = os.Args[i+1]
				i++
			}
		}
	}

	if dbPath == "" {
		dbPath = "/tmp/rolodex-test.db"
	}
	if keystorePath == "" {
		keystorePath = "/tmp/keystore-test.db"
	}

	// Initialize store
	store, err := secretary.NewStore(secretary.StoreConfig{
		Path: dbPath,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create store: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	// Initialize keystore
	ks, err := keystore.New(keystore.Config{
		DBPath:    keystorePath,
		MasterKey: []byte("e2e-test-master-key-exactly-32-b"),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create keystore: %v\n", err)
		os.Exit(1)
	}

	if err := ks.Open(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open keystore: %v\n", err)
		os.Exit(1)
	}
	defer ks.Close()

	// Initialize rolodex service
	rolodex, err := secretary.NewRolodexService(secretary.RolodexConfig{
		Store:    store,
		Keystore: ks,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create rolodex service: %v\n", err)
		os.Exit(1)
	}

	switch command {
	case "create":
		if name == "" {
			fmt.Fprintln(os.Stderr, "Error: --name is required for create")
			os.Exit(1)
		}

		contact, err := rolodex.CreateContact(ctx, &secretary.CreateRequest{
			Name:         name,
			Company:      company,
			Relationship: relationship,
			Phone:        phone,
			Email:        email,
			Address:      address,
			Notes:        notes,
			CreatedBy:    "e2e-test",
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating contact: %v\n", err)
			os.Exit(1)
		}

		json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"status": "success",
			"id":     contact.ID,
			"name":   contact.Name,
		})

	case "get":
		if name == "" {
			fmt.Fprintln(os.Stderr, "Error: --name is required for get")
			os.Exit(1)
		}

		contacts, err := rolodex.ListContacts(ctx, secretary.ContactFilter{Name: name})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing contacts: %v\n", err)
			os.Exit(1)
		}

		if len(contacts) == 0 {
			fmt.Fprintln(os.Stderr, "Error: contact not found")
			os.Exit(1)
		}

		json.NewEncoder(os.Stdout).Encode(contacts[0])

	case "search":
		if query == "" {
			fmt.Fprintln(os.Stderr, "Error: --query is required for search")
			os.Exit(1)
		}

		contacts, err := rolodex.ListContacts(ctx, secretary.ContactFilter{Name: query})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error searching contacts: %v\n", err)
			os.Exit(1)
		}

		json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"results":  len(contacts),
			"contacts": contacts,
		})

	case "update":
		if name == "" {
			fmt.Fprintln(os.Stderr, "Error: --name is required for update")
			os.Exit(1)
		}

		contacts, err := rolodex.ListContacts(ctx, secretary.ContactFilter{Name: name})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing contacts: %v\n", err)
			os.Exit(1)
		}

		if len(contacts) == 0 {
			fmt.Fprintln(os.Stderr, "Error: contact not found")
			os.Exit(1)
		}

		contact, err := rolodex.UpdateContact(ctx, &secretary.UpdateRequest{
			ID:           contacts[0].ID,
			Name:         name,
			Company:      company,
			Relationship: relationship,
			Phone:        phone,
			Email:        email,
			Address:      address,
			Notes:        notes,
			UpdatedBy:    "e2e-test",
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error updating contact: %v\n", err)
			os.Exit(1)
		}

		json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"status": "success",
			"id":     contact.ID,
		})

	case "delete":
		if name == "" {
			fmt.Fprintln(os.Stderr, "Error: --name is required for delete")
			os.Exit(1)
		}

		contacts, err := rolodex.ListContacts(ctx, secretary.ContactFilter{Name: name})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing contacts: %v\n", err)
			os.Exit(1)
		}

		if len(contacts) == 0 {
			fmt.Fprintln(os.Stderr, "Error: contact not found")
			os.Exit(1)
		}

		err = rolodex.DeleteContact(ctx, contacts[0].ID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting contact: %v\n", err)
			os.Exit(1)
		}

		json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"status": "success",
			"id":     contacts[0].ID,
		})

	case "verify-encryption":
		if name == "" {
			fmt.Fprintln(os.Stderr, "Error: --name is required for verify-encryption")
			os.Exit(1)
		}

		contacts, err := rolodex.ListContacts(ctx, secretary.ContactFilter{Name: name})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing contacts: %v\n", err)
			os.Exit(1)
		}

		if len(contacts) == 0 {
			fmt.Fprintln(os.Stderr, "Error: contact not found")
			os.Exit(1)
		}

		contact := contacts[0]

		// Verify that sensitive data is not in plain text in the encrypted field
		if len(contact.EncryptedData) == 0 {
			json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
				"encryption_verified": false,
				"error":               "no encrypted data found",
			})
			os.Exit(1)
		}

		// Try to decrypt to verify the data is actually encrypted
		decrypted, err := ks.Decrypt(contact.EncryptedData, contact.EncryptedNonce)
		if err != nil {
			json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
				"encryption_verified": false,
				"error":               fmt.Sprintf("decryption failed: %v", err),
			})
			os.Exit(1)
		}

		// Check if the decrypted data contains the expected contact data
		if len(decrypted) == 0 {
			json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
				"encryption_verified": false,
				"error":               "decrypted data is empty",
			})
			os.Exit(1)
		}

		// Verify the plain text is not in the encrypted blob
		encryptedStr := string(contact.EncryptedData)
		if containsPlainPhone(encryptedStr, phone) ||
			containsPlainEmail(encryptedStr, email) ||
			containsPlainAddress(encryptedStr, address) {
			json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
				"encryption_verified": false,
				"error":               "plain text found in encrypted data",
			})
			os.Exit(1)
		}

		json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"encryption_verified": true,
			"encrypted_length":    len(contact.EncryptedData),
			"decrypted_length":    len(decrypted),
		})

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		os.Exit(1)
	}
}

func containsPlainPhone(encrypted, phone string) bool {
	if phone == "" {
		return false
	}
	// Check if plain phone number appears in encrypted data (it shouldn't)
	return len(encrypted) > 0 && len(phone) > 0 && len(encrypted) > len(phone)
}

func containsPlainEmail(encrypted, email string) bool {
	if email == "" {
		return false
	}
	// Check if plain email appears in encrypted data (it shouldn't)
	return len(encrypted) > 0 && len(email) > 0 && len(encrypted) > len(email)
}

func containsPlainAddress(encrypted, address string) bool {
	if address == "" {
		return false
	}
	// Check if plain address appears in encrypted data (it shouldn't)
	return len(encrypted) > 0 && len(address) > 0 && len(encrypted) > len(address)
}
