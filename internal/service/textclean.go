package service

import "regexp"

var (
	legacyColorCodeRe = regexp.MustCompile(`[§&][0-9A-FK-ORXa-fk-orx]`)
	hexColorCodeRe    = regexp.MustCompile(`#[0-9A-Fa-f]{6}`)
)

// CleanReason strips Minecraft chat formatting codes (§x/&x and #RRGGBB) from punishment reason text.
func CleanReason(s string) string {
	s = legacyColorCodeRe.ReplaceAllString(s, "")
	s = hexColorCodeRe.ReplaceAllString(s, "")
	return s
}
