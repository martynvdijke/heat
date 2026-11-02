package app

import (
	"os"
	"strings"
)

// LegalConfig is the operator identity that the published privacy policy,
// terms and account-deletion page are allowed to assert. Values come from the
// environment so a self-hosted deployment can name itself without a code
// change. Empty values fall back to neutral wording rather than inventing a
// controller.
type LegalConfig struct {
	Controller     string
	ContactEmail   string
	ContactAddress string
	HostingRegion  string
	BaseURL        string
}

// LoadLegalConfig reads LEGAL_* and PUBLIC_BASE_URL. The defaults are
// deliberately generic: a deployment that does not configure a contact address
// must not pretend it has one.
func LoadLegalConfig() LegalConfig {
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("PUBLIC_BASE_URL")), "/")
	if base == "" {
		base = strings.TrimRight(strings.TrimSpace(os.Getenv("LEGAL_BASE_URL")), "/")
	}
	if base == "" {
		base = "https://heat.vandijke.xyz"
	}
	c := LegalConfig{
		Controller:     strings.TrimSpace(os.Getenv("LEGAL_CONTROLLER_NAME")),
		ContactEmail:   strings.TrimSpace(os.Getenv("LEGAL_CONTACT_EMAIL")),
		ContactAddress: strings.TrimSpace(os.Getenv("LEGAL_CONTACT_ADDRESS")),
		HostingRegion:  strings.TrimSpace(os.Getenv("LEGAL_HOSTING_REGION")),
		BaseURL:        base,
	}
	if c.Controller == "" {
		c.Controller = "the operator of this HEAT instance"
	}
	if c.HostingRegion == "" {
		c.HostingRegion = "the European Union"
	}
	return c
}
