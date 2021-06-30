package serializer

import (
	"strings"

	"github.com/CanalTP/gonavitia"
	"github.com/CanalTP/gonavitia/pbnavitia"
	"github.com/golang/protobuf/proto"
)

func (s *Serializer) NewJourneysReponse(pb *pbnavitia.Response) *gonavitia.JourneysResponse {
	if pb == nil {
		return nil
	}
	r := gonavitia.JourneysResponse{}
	for _, pb_journey := range pb.Journeys {
		r.Journeys = append(r.Journeys, s.NewJourney(pb_journey))
	}
	return &r
}

func (s *Serializer) NewJourney(pb *pbnavitia.Journey) *gonavitia.Journey {
	if pb == nil {
		return nil
	}
	journey := gonavitia.Journey{
		From:              s.NewPlace(pb.Origin),
		To:                s.NewPlace(pb.Destination),
		Duration:          pb.GetDuration(),
		NbTransfers:       pb.GetNbTransfers(),
		DepartureDateTime: s.NewNavitiaDatetime(int64(pb.GetDepartureDateTime())),
		ArrivalDateTime:   s.NewNavitiaDatetime(int64(pb.GetArrivalDateTime())),
		RequestedDateTime: s.NewNavitiaDatetime(int64(pb.GetRequestedDateTime())),
		Status:            pb.GetMostSeriousDisruptionEffect(),
		Durations:         s.NewDurations(pb.Durations),
		Distances:         s.NewDistances(pb.Distances),
		Tags:              make([]string, 0),
	}
	for _, pb_section := range pb.Sections {
		journey.Sections = append(journey.Sections, s.NewSection(pb_section))
	}
	return &journey
}

func (s *Serializer) NewSection(pb *pbnavitia.Section) *gonavitia.Section {
	if pb == nil {
		return nil
	}
	var mode *string
	if sn := pb.StreetNetwork; sn != nil {
		m := strings.ToLower(sn.Mode.String())
		mode = &m
	}
	var transferType *string
	if pb.TransferType != nil {
		t := strings.ToLower(pb.TransferType.String())
		transferType = &t
	}
	section := gonavitia.Section{
		Id:                pb.GetId(),
		From:              s.NewPlace(pb.Origin),
		To:                s.NewPlace(pb.Destination),
		DepartureDateTime: s.NewNavitiaDatetime(int64(pb.GetBeginDateTime())),
		ArrivalDateTime:   s.NewNavitiaDatetime(int64(pb.GetEndDateTime())),
		Duration:          pb.GetDuration(),
		Type:              strings.ToLower(pb.GetType().String()),
		GeoJson:           s.NewGeoJson(pb),
		Mode:              mode,
		TransferType:      transferType,
		DisplayInfo:       s.NewPtDisplayInfoForVJ(pb.PtDisplayInformations),
		Co2Emission:       s.NewCo2Emission(pb.Co2Emission),
		AdditionalInfo:    s.NewAdditionalInformations(pb.AdditionalInformations),
		Links:             s.NewLinksFromUris(pb.PtDisplayInformations),
	}

	return &section
}

func (s *Serializer) NewDurations(pb *pbnavitia.Durations) *gonavitia.Durations {
	if pb == nil {
		return nil
	}
	durations := gonavitia.Durations{
		Total:       pb.GetTotal(),
		Walking:     pb.GetWalking(),
		Bike:        pb.GetBike(),
		Car:         pb.GetCar(),
		Ridesharing: pb.GetRidesharing(),
	}
	return &durations
}

func (s *Serializer) NewDistances(pb *pbnavitia.Distances) *gonavitia.Distances {
	if pb == nil {
		return nil
	}
	distances := gonavitia.Distances{
		Walking:     pb.GetWalking(),
		Bike:        pb.GetBike(),
		Car:         pb.GetCar(),
		Ridesharing: pb.GetRidesharing(),
	}
	return &distances
}

func (s *Serializer) NewCo2Emission(pb *pbnavitia.Co2Emission) *gonavitia.Amount {
	if pb == nil {
		return nil
	}
	co2 := gonavitia.Amount{
		Value: pb.GetValue(),
		Unit:  pb.GetUnit(),
	}
	return &co2

}

func (s *Serializer) NewLinksFromUris(pb *pbnavitia.PtDisplayInfo) []gonavitia.Link {
	if pb == nil || pb.Uris == nil {
		return nil
	}
	uris := pb.Uris
	res := make([]gonavitia.Link, 0)
	res = s.appendLinksFromUri(uris.Company, "company", &res)
	res = s.appendLinksFromUri(uris.VehicleJourney, "vehicle_journey", &res)
	res = s.appendLinksFromUri(uris.Line, "line", &res)
	res = s.appendLinksFromUri(uris.Route, "route", &res)
	res = s.appendLinksFromUri(uris.CommercialMode, "commercial_mode", &res)
	res = s.appendLinksFromUri(uris.PhysicalMode, "physical_mode", &res)
	res = s.appendLinksFromUri(uris.Network, "network", &res)
	res = s.appendLinksFromUri(uris.Note, "note", &res)
	res = s.appendLinksFromUri(uris.JourneyPattern, "journey_pattern", &res)
	return res
}

func (s *Serializer) appendLinksFromUri(pb *string, typ string, links *[]gonavitia.Link) []gonavitia.Link {
	if pb == nil {
		return *links
	}
	return append(*links, gonavitia.Link{Id: pb, Type: proto.String(typ)})
}
