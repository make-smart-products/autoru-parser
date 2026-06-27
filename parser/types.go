package parser

// Listing contains parsed data from an auto.ru car sale page.
type Listing struct {
	URL            string            `json:"url"`
	OfferID        string            `json:"offer_id"`
	Title          string            `json:"title"`
	Price          int64             `json:"price,omitempty"`
	PriceFormatted string            `json:"price_formatted,omitempty"`
	Description    string            `json:"description,omitempty"`
	Location       string            `json:"location,omitempty"`
	Seller         string            `json:"seller,omitempty"`
	Comment        string            `json:"comment,omitempty"`
	Ownership      map[string]string `json:"ownership,omitempty"`
	Characteristics map[string]string `json:"characteristics,omitempty"`
	TechSpecs      []SpecGroup       `json:"tech_specs,omitempty"`
	Equipment      []EquipmentGroup  `json:"equipment,omitempty"`
	Photos         []string          `json:"photos"`
	CatalogIDs     *CatalogIDs       `json:"catalog_ids,omitempty"`
}

type CatalogIDs struct {
	Mark            string `json:"mark,omitempty"`
	Model           string `json:"model,omitempty"`
	GenerationID    int    `json:"generation_id,omitempty"`
	ConfigurationID int    `json:"configuration_id"`
	ComplectationID int    `json:"complectation_id"`
	TechParamID     int    `json:"tech_param_id"`
}

type SpecGroup struct {
	ID     string     `json:"id"`
	Name   string     `json:"name,omitempty"`
	Fields []SpecItem `json:"fields"`
}

type SpecItem struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"`
	Units string `json:"units,omitempty"`
}

type EquipmentGroup struct {
	Name  string   `json:"name"`
	Items []string `json:"items"`
}

// AllSpecs returns a flat map of ownership, characteristics, and catalog tech fields.
func (l *Listing) AllSpecs() map[string]string {
	if l == nil {
		return nil
	}
	out := make(map[string]string, len(l.Ownership)+len(l.Characteristics)+32)
	for k, v := range l.Ownership {
		out["Владение: "+k] = v
	}
	for k, v := range l.Characteristics {
		out[k] = v
	}
	for _, group := range l.TechSpecs {
		prefix := group.Name
		if prefix == "" || prefix == "NO_GROUP" {
			prefix = ""
		}
		for _, field := range group.Fields {
			key := field.Name
			if prefix != "" {
				key = prefix + ": " + field.Name
			}
			out[key] = field.Value
		}
	}
	return out
}
