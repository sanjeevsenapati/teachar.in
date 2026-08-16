package models

// Store represents a physical TEACHAR outlet.
type Store struct {
	ID int64
}

// CafeSettings represents configurable store & announcement bar settings.
type CafeSettings struct {
	StoreName           string `json:"store_name"`
	StoreAddress        string `json:"store_address"`
	BrewingHours        string `json:"brewing_hours"`
	StorePhone          string `json:"store_phone"`
	CurrencySymbol      string `json:"currency_symbol"`
	AnnouncementEnabled bool   `json:"announcement_enabled"`
	AnnouncementText    string `json:"announcement_text"`
	AnnouncementPhone   string `json:"announcement_phone"`
}
