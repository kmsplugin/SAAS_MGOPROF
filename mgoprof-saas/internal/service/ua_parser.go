package service

import (
	"strings"

	"mgoprof-saas/internal/model"
)

// ParseUserAgent extracts device type, OS and browser from a User-Agent string.
// It uses simple string matching — no external library required.
func ParseUserAgent(ua string) model.DeviceInfo {
	if ua == "" {
		return model.DeviceInfo{DeviceType: "unknown", OSName: "unknown", BrowserName: "unknown"}
	}

	lower := strings.ToLower(ua)

	return model.DeviceInfo{
		DeviceType:  detectDeviceType(lower),
		OSName:      detectOS(lower),
		BrowserName: detectBrowser(lower),
	}
}

func detectDeviceType(lower string) string {
	// Bots / crawlers first
	for _, kw := range []string{"bot", "crawler", "spider", "slurp", "archiver", "facebookexternalhit"} {
		if strings.Contains(lower, kw) {
			return "bot"
		}
	}

	// Tablets — must be checked before "mobile"
	if strings.Contains(lower, "ipad") {
		return "tablet"
	}
	if strings.Contains(lower, "tablet") {
		return "tablet"
	}
	if strings.Contains(lower, "android") && !strings.Contains(lower, "mobile") {
		return "tablet"
	}

	// Phones
	for _, kw := range []string{"iphone", "ipod", "mobile", "blackberry", "windows phone", "opera mini", "opera mobi"} {
		if strings.Contains(lower, kw) {
			return "mobile"
		}
	}

	return "desktop"
}

func detectOS(lower string) string {
	switch {
	case strings.Contains(lower, "windows phone"):
		return "Windows Phone"
	case strings.Contains(lower, "windows"):
		return "Windows"
	case strings.Contains(lower, "iphone") || strings.Contains(lower, "ipad") || strings.Contains(lower, "ipod"):
		return "iOS"
	case strings.Contains(lower, "mac os") || strings.Contains(lower, "macintosh"):
		return "macOS"
	case strings.Contains(lower, "android"):
		return "Android"
	case strings.Contains(lower, "linux"):
		return "Linux"
	case strings.Contains(lower, "cros"):
		return "ChromeOS"
	case strings.Contains(lower, "freebsd") || strings.Contains(lower, "openbsd") || strings.Contains(lower, "netbsd"):
		return "BSD"
	default:
		return "unknown"
	}
}

func detectBrowser(lower string) string {
	// Order matters — more specific checks first.
	switch {
	case strings.Contains(lower, "yabrowser"):
		return "YandexBrowser"
	case strings.Contains(lower, "edg/") || strings.Contains(lower, "edge/"):
		return "Edge"
	case strings.Contains(lower, "opr/") || strings.Contains(lower, "opera"):
		return "Opera"
	case strings.Contains(lower, "samsungbrowser"):
		return "Samsung Internet"
	case strings.Contains(lower, "ucbrowser"):
		return "UC Browser"
	case strings.Contains(lower, "miuibrowser"):
		return "MIUI Browser"
	case strings.Contains(lower, "chromium"):
		return "Chromium"
	// Chrome must come after Edge/Opera/Samsung because they all include "chrome"
	case strings.Contains(lower, "chrome"):
		return "Chrome"
	case strings.Contains(lower, "safari") && !strings.Contains(lower, "chrome"):
		return "Safari"
	case strings.Contains(lower, "firefox"):
		return "Firefox"
	case strings.Contains(lower, "msie") || strings.Contains(lower, "trident"):
		return "IE"
	default:
		return "other"
	}
}
