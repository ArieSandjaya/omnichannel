package domain

import (
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
)

type TenantTheme struct {
	PrimaryColor   string `json:"primary_color"`
	SecondaryColor string `json:"secondary_color"`
	HeroText       string `json:"hero_text"`
	FontFamily     string `json:"font_family"`
}

type TenantStore struct {
	Address string `json:"address"`
	Phone   string `json:"phone"`
	Email   string `json:"email"`
}

type TenantSettings struct {
	Theme TenantTheme `json:"theme"`
	Store TenantStore `json:"store"`
}

type Tenant struct {
	ID                 uuid.UUID      `json:"id"`
	Name               string         `json:"name"`
	Slug               string         `json:"slug"`
	BusinessType       string         `json:"business_type"`
	SubscriptionPlan   string         `json:"subscription_plan"`
	SubscriptionStatus string         `json:"subscription_status"`
	LogoURL            string         `json:"logo_url"`
	Currency           string         `json:"currency"`
	Timezone           string         `json:"timezone"`
	Settings           TenantSettings `json:"settings"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
}

var hexColorRegex = regexp.MustCompile(`^#[0-9a-fA-F]{3,8}$`)

func sanitizeColor(s, fallback string) string {
	if hexColorRegex.MatchString(s) {
		return s
	}
	return fallback
}

func (t Tenant) GetPrimaryColor() string {
	return sanitizeColor(t.Settings.Theme.PrimaryColor, "#6366f1")
}

func (t Tenant) GetSecondaryColor() string {
	return sanitizeColor(t.Settings.Theme.SecondaryColor, "#ec4899")
}

func (t Tenant) GetHeroText() string {
	if t.Settings.Theme.HeroText != "" {
		return t.Settings.Theme.HeroText
	}
	return fmt.Sprintf("Selamat Datang di %s", t.Name)
}

func (t Tenant) GetFontFamily() string {
	if t.Settings.Theme.FontFamily != "" {
		return t.Settings.Theme.FontFamily
	}
	return "Inter, sans-serif"
}

// ThemeCSS returns the inline <style> block for CSS custom properties.
// Use with templ.Raw() in storefront layout.
func (t Tenant) ThemeCSS() string {
	return fmt.Sprintf(
		`:root{--color-primary:%s;--color-secondary:%s;--font-family:%s}`,
		t.GetPrimaryColor(),
		t.GetSecondaryColor(),
		t.GetFontFamily(),
	)
}
