package auth

type Session struct {
	ID         int64  `json:"id"`
	TokenHash  []byte `json:"-"`
	UserID     int64  `json:"userId"`
	IPAddress  string `json:"ipAddress"`
	LastUsedIP string `json:"lastUsedIP"`
	UserAgent  string `json:"userAgent"`
	ExpiresAt  int64  `json:"expiresAt"`
	LastUsedAt int64  `json:"lastUsedAt"`
	CreatedAt  int64  `json:"createdAt"`
	UpdatedAt  int64  `json:"updatedAt"`
}
