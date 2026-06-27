package parser

import (
	"encoding/json"
	"fmt"
)

type techInfoResponse struct {
	Status string `json:"status"`
	Error  string `json:"error"`
	Data   struct {
		TechInfoGroup []struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			Entity []struct {
				ID    string `json:"id"`
				Name  string `json:"name"`
				Value string `json:"value"`
				Units string `json:"units"`
			} `json:"entity"`
		} `json:"tech_info_group"`
	} `json:"data"`
}

func parseTechInfo(raw string) ([]SpecGroup, error) {
	var resp techInfoResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return nil, err
	}
	if resp.Status == "ERROR" {
		return nil, fmt.Errorf("api error: %s", resp.Error)
	}

	var groups []SpecGroup
	for _, g := range resp.Data.TechInfoGroup {
		group := SpecGroup{ID: g.ID, Name: g.Name}
		for _, item := range g.Entity {
			value := item.Value
			if item.Units != "" {
				value = fmt.Sprintf("%s %s", value, item.Units)
			}
			group.Fields = append(group.Fields, SpecItem{
				ID:    item.ID,
				Name:  item.Name,
				Value: value,
				Units: item.Units,
			})
		}
		if len(group.Fields) > 0 {
			groups = append(groups, group)
		}
	}
	return groups, nil
}

type namedItems struct {
	Name   string `json:"name"`
	Values []struct {
		Name string `json:"name"`
	} `json:"values"`
	Entities []struct {
		Name string `json:"name"`
	} `json:"entities"`
	Items []string `json:"items"`
}

type equipmentResponse struct {
	Status string `json:"status"`
	Error  string `json:"error"`
	Data   struct {
		Equipment    []namedItems `json:"equipment"`
		OptionGroups []namedItems `json:"option_groups"`
		Options      []struct {
			Group string `json:"group"`
			Name  string `json:"name"`
		} `json:"options"`
	} `json:"data"`
}

func parseEquipment(raw string) ([]EquipmentGroup, error) {
	var resp equipmentResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return nil, err
	}
	if resp.Status == "ERROR" {
		return nil, fmt.Errorf("api error: %s", resp.Error)
	}

	groups := equipmentFromBlocks(resp.Data.Equipment)
	if len(groups) == 0 {
		groups = equipmentFromBlocks(resp.Data.OptionGroups)
	}
	if len(groups) == 0 && len(resp.Data.Options) > 0 {
		byGroup := map[string][]string{}
		for _, opt := range resp.Data.Options {
			byGroup[opt.Group] = append(byGroup[opt.Group], opt.Name)
		}
		for name, items := range byGroup {
			groups = append(groups, EquipmentGroup{Name: name, Items: items})
		}
	}
	return groups, nil
}

func equipmentFromBlocks(blocks []namedItems) []EquipmentGroup {
	var groups []EquipmentGroup
	for _, block := range blocks {
		var items []string
		for _, v := range block.Values {
			if v.Name != "" {
				items = append(items, v.Name)
			}
		}
		for _, e := range block.Entities {
			if e.Name != "" {
				items = append(items, e.Name)
			}
		}
		items = append(items, block.Items...)
		if len(items) > 0 {
			groups = append(groups, EquipmentGroup{Name: block.Name, Items: items})
		}
	}
	return groups
}
