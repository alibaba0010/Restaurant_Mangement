package address

import (
	"context"

	"github.com/alibaba0010/postgres-api/internal/common/errors"
	"github.com/alibaba0010/postgres-api/internal/common/logger"

	"github.com/Boostport/address"
	"github.com/codingsince1985/geo-golang"
	"github.com/codingsince1985/geo-golang/openstreetmap"
	"go.uber.org/zap"
)

// Coordinates represents a geographical location.
type Coordinates struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// AddressInput represents a simplified physical address structure used for formatting and validation.
type AddressInput struct {
	Address  string `json:"address" validate:"required"`
	City     string `json:"city" validate:"required"`
	Country  string `json:"country" validate:"required"`
	PostCode string `json:"post_code,omitempty"`
}

// Address Service defines the interface for address related operations.
type AddressService interface {
	// Geocode converts a free-form address string to coordinates and returns the formatted address.
	Geocode(ctx context.Context, fullAddress string) (*Coordinates, string, *errors.AppError)

	// ReverseGeocode converts coordinates to a formatted address string.
	ReverseGeocode(ctx context.Context, lat, lng float64) (string, *errors.AppError)

	// Format validates and formats an address structure into a formatted string.
	Format(addr AddressInput) (string, *errors.AppError)

	// ProcessAddress formats the address and geocodes it to get coordinates.
	ProcessAddress(ctx context.Context, addrInput *AddressInput) (string, float64, float64, *errors.AppError)
}

type addressService struct {
	geocoder geo.Geocoder
}

// NewService creates a new instance of the address service.
// Currently uses OpenStreetMap. In a production environment, this should ideally be configurable
// (e.g., Google Maps, MapQuest) via options or config.
func NewService() AddressService {
	return &addressService{
		geocoder: openstreetmap.Geocoder(),
	}
}

func (s *addressService) Geocode(ctx context.Context, fullAddress string) (*Coordinates, string, *errors.AppError) {
	if fullAddress == "" {
		return nil, "", errors.BadRequestError("Address field should not be empty")
	}

	location, err := s.geocoder.Geocode(fullAddress)
	if err != nil {
		logger.Log.Error("geocoding failed", zap.Error(err))
		return nil, "", errors.ValidationError("geocoding failed: " + err.Error())
	}

	if location == nil {
		return nil, "", errors.NotFoundError("address not found")
	}

	// Reverse geocode to get formatted address
	formattedAddress, err := s.ReverseGeocode(ctx, location.Lat, location.Lng)
	if err != nil {
		// Fallback to original string if reverse geocoding fails
		formattedAddress = fullAddress
	}

	return &Coordinates{
		Latitude:  location.Lat,
		Longitude: location.Lng,
	}, formattedAddress, nil
}

func (s *addressService) ReverseGeocode(ctx context.Context, lat, lng float64) (string, *errors.AppError) {
	addr, err := s.geocoder.ReverseGeocode(lat, lng)
	if err != nil {
		logger.Log.Error("reverse geocoding failed", zap.Error(err))
		return "", errors.ValidationError("reverse geocoding failed: " + err.Error())
	}
	
	if addr == nil || addr.FormattedAddress == "" {
		return "", errors.NotFoundError("Address not found for the given coordinates")
	}

	return addr.FormattedAddress, nil
}

func (s *addressService) Format(addr AddressInput) (string, *errors.AppError) {
	// Convert AddressInput to Boostport/address Address struct
	bpAddr := address.Address{
		StreetAddress: []string{addr.Address},
		Locality:      addr.City,
		PostCode:      addr.PostCode,
		Country:       addr.Country,
	}

	// Validate the address
	if err := address.Validate(bpAddr); err != nil {
		// Fallback to simple formatting if validation fails (e.g., country name vs code)
		formatted := addr.Address + ", " + addr.City
		if addr.PostCode != "" {
			formatted += ", " + addr.PostCode
		}
		formatted += ", " + addr.Country
		return formatted, nil
	}

	// Format the address using PostalLabelFormatter
	formatter := address.PostalLabelFormatter{
		Output: address.StringOutputter{},
	}
	output := formatter.Format(bpAddr, "en")

	return output, nil
}

// ProcessAddress formats the address and geocodes it to get coordinates.
// Returns formatted address, latitude, longitude, and error.
func (s *addressService) ProcessAddress(ctx context.Context, addrInput *AddressInput) (string, float64, float64, *errors.AppError) {
	// Format the address structure
	fmtStr, err := s.Format(*addrInput)
	if err != nil {
		return "", 0, 0, err
	}

	// Geocode the formatted string to get coordinates
	coords, _, err := s.Geocode(ctx, fmtStr)
	if err != nil {
		return "", 0, 0, err
	}

	return fmtStr, coords.Latitude, coords.Longitude, nil
}