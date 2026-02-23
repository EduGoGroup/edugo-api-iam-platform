package dto

// MenuItemDTO represents a menu item
type MenuItemDTO struct {
	Key         string            `json:"key"`
	DisplayName string            `json:"display_name"`
	Icon        string            `json:"icon,omitempty"`
	Scope       string            `json:"scope"`
	SortOrder   int               `json:"sort_order"`
	Permissions []string          `json:"permissions,omitempty"`
	Screens     map[string]string `json:"screens,omitempty"`
	Children    []MenuItemDTO     `json:"children"`
}

// MenuResponse wraps a list of menu items
type MenuResponse struct {
	Items []MenuItemDTO `json:"items"`
}
