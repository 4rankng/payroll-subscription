// Feature: audit-log-revamp, Property 11: Audit message contains entity-specific information
package domain

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// genNonEmptyAlphaString generates a random non-empty alphabetic string.
func genNonEmptyAlphaString() gopter.Gen {
	return gen.AlphaString().SuchThat(func(s string) bool {
		return len(s) > 0
	})
}

// TestProperty11_PayrateAuditMessageContainsProjectName verifies that NewPayrateCreatedEvent
// produces an AuditMessage containing the project name, with Action=CREATE and
// EntityType=payrate.
//
// Validates: Requirements 6.3, 3.12
func TestProperty11_PayrateAuditMessageContainsProjectName(t *testing.T) {
	parameters := gopter.DefaultTestParametersWithSeed(11002)
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("PayrateCreatedEvent AuditMessage contains project name", prop.ForAll(
		func(projectName string) bool {
			payrate := &Payrate{
				ID:        1,
				ProjectID: 1,
				FromDate:  time.Now(),
				Project: Project{
					ID:   1,
					Name: projectName,
				},
			}
			event := NewPayrateCreatedEvent(context.Background(), payrate)

			msg := event.GetAuditMessage()
			return msg != "" &&
				strings.Contains(msg, projectName) &&
				event.GetAction() == AuditActionCreate &&
				event.GetEntityType() == EntityTypePayrate
		},
		genNonEmptyAlphaString(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty11_AssetAuditMessageContainsFilename verifies that NewAssetCreatedEvent
// produces an AuditMessage containing the asset's Filename, with Action=CREATE and
// EntityType=asset.
//
// Validates: Requirements 6.7, 3.12
func TestProperty11_AssetAuditMessageContainsFilename(t *testing.T) {
	parameters := gopter.DefaultTestParametersWithSeed(11003)
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("AssetCreatedEvent AuditMessage contains Filename", prop.ForAll(
		func(filename string) bool {
			asset := &Asset{
				ID:       1,
				Filename: filename,
				FilePath: "/uploads/" + filename,
			}
			event := NewAssetCreatedEvent(context.Background(), asset)

			msg := event.GetAuditMessage()
			return msg != "" &&
				strings.Contains(msg, filename) &&
				event.GetAction() == AuditActionCreate &&
				event.GetEntityType() == EntityTypeAsset
		},
		genNonEmptyAlphaString(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty11_SettingsAuditMessageContainsKey verifies that NewSettingsCreatedEvent
// produces an AuditMessage containing the settings key, with Action=CREATE and
// EntityType=settings.
//
// Validates: Requirements 6.10, 3.12
func TestProperty11_SettingsAuditMessageContainsKey(t *testing.T) {
	parameters := gopter.DefaultTestParametersWithSeed(11005)
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("SettingsCreatedEvent AuditMessage contains settings key", prop.ForAll(
		func(key string) bool {
			event := NewSettingsCreatedEvent(context.Background(), key, 1)

			msg := event.GetAuditMessage()
			return msg != "" &&
				strings.Contains(msg, key) &&
				event.GetAction() == AuditActionCreate &&
				event.GetEntityType() == EntityTypeSettings
		},
		genNonEmptyAlphaString(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
