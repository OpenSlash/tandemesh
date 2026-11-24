package domain

// MobilePairing is a short-lived, single-use code used to pair a mobile device.
type MobilePairing struct {
	ID          string
	CodeHash    string
	UserID      string
	WorkspaceID string
	ExpiresAtMS int64
	UsedAtMS    int64
	CreatedAtMS int64
}

// MobileToken is a bearer credential held by a paired mobile device.
type MobileToken struct {
	ID          string
	UserID      string
	TokenHash   string
	DeviceName  string
	Platform    string
	CreatedAtMS int64
	LastUsedMS  int64
	RevokedAtMS int64
}
