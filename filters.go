package gormungandr

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

type FilterURI struct {
	Filters []string
	API     string
}

func protect(id string) string {
	return "\"" + strings.Replace(id, "\"", "\\\"", -1) + "\""
}

func convertToFilter(collection, value string) (string, error) {
	typ, found := collectionToType[collection]
	if !found {
		return "", fmt.Errorf("type %s unkwown", collection)
	}
	if typ == "coord" {
		return "", errors.New("coord aren't implemented yet")
	}
	return fmt.Sprintf("%s.uri=%s", typ, protect(value)), nil
}

func ParsePath(path string) (FilterURI, error) {
	path = strings.Trim(path, "/")
	if len(path) < 1 {
		return FilterURI{}, errors.New("path is empty")
	}
	paths := strings.Split(path, "/")
	var typ string
	var result FilterURI
	//TODO reverse to get the API first, it's mandatory for some case like coord
	for i, v := range paths {
		if i%2 == 0 {
			typ = v
			result.API = v
		} else {
			f, err := convertToFilter(typ, v)
			if err != nil {
				return FilterURI{}, err
			}
			result.Filters = append(result.Filters, f)
		}
	}
	return result, nil
}

var (
	collectionToType = map[string]string{
		"stop_points":            "stop_point",
		"routes":                 "route",
		"networks":               "network",
		"commercial_modes":       "commercial_mode",
		"physical_modes":         "physical_mode",
		"companies":              "company",
		"stop_areas":             "stop_area",
		"lines":                  "line",
		"line_groups":            "line_group",
		"addresses":              "address",
		"coords":                 "coord",
		"coord":                  "coord",
		"journey_pattern_points": "journey_pattern_point",
		"journey_patterns":       "journey_pattern",
		"pois":                   "poi",
		"poi_types":              "poi_type",
		"connections":            "connection",
		"vehicle_journeys":       "vehicle_journey",
		"disruptions":            "disruption",
		"trips":                  "trip",
		"contributors":           "contributor",
		"datasets":               "dataset",
	}
)
