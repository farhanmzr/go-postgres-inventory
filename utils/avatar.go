package utils

import (
	"net/url"
	"strings"
)

// Default avatar DiceBear, style fixed: gradientLinear, size 256 PNG
func DefaultAvatar(fullName string) string {
	seed := url.QueryEscape(fullName)
	return "https://api.dicebear.com/7.x/initials/png?seed=" + seed +
		"&size=256&backgroundType=gradientLinear"
}

// Jika URL Cloudinary, selipkan transformasi agar kecil (256px, auto-compress)
func CloudinaryThumb256(secureURL string) string {
	if secureURL == "" { return secureURL }
	if !strings.Contains(secureURL, "/image/upload/") {
		return secureURL // bukan Cloudinary
	}
	return strings.Replace(
		secureURL,
		"/image/upload/",
		"/image/upload/f_auto,q_auto,w_256,h_256,c_fill,g_face,r_max/",
		1,
	)
}
