package application

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/hrm/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/google/uuid"
)

const (
	defaultGeofenceRadiusMeters = 100
	// Work starts at 09:00 local time; clocking in at or after it is "Late".
	workStartHour = 9
)

// attendanceZone is the timezone attendance dates and lateness are judged in,
// independent of the server's own timezone.
var attendanceZone = func() *time.Location {
	if loc, err := time.LoadLocation("Asia/Jakarta"); err == nil {
		return loc
	}
	return time.FixedZone("WIB", 7*60*60)
}()

// haversineMeters is the great-circle distance between two coordinates.
func haversineMeters(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371000.0
	toRad := func(d float64) float64 { return d * math.Pi / 180 }
	dLat, dLon := toRad(lat2-lat1), toRad(lon2-lon1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Sin(dLon/2)*math.Sin(dLon/2)
	return earthRadius * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

// matchLocation checks the coordinates against the configured attendance
// locations. With no active locations, geofencing is off and (nil, nil) is
// returned. Otherwise the coordinates are required and must fall inside one.
func (uc *hrmUseCase) matchLocation(ctx context.Context, lat, lon *float64) (*domain.AttendanceLocation, error) {
	locs, err := uc.repo.ListAttendanceLocations(ctx, true)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load attendance locations")
	}
	if len(locs) == 0 {
		return nil, nil
	}
	if lat == nil || lon == nil {
		return nil, apperrors.NewBadRequest("Latitude and longitude are required to clock in at a configured location")
	}
	nearest, nearestDist := &locs[0], math.MaxFloat64
	for i := range locs {
		d := haversineMeters(*lat, *lon, locs[i].Latitude, locs[i].Longitude)
		if d <= float64(locs[i].RadiusMeters) {
			return &locs[i], nil
		}
		if d < nearestDist {
			nearest, nearestDist = &locs[i], d
		}
	}
	return nil, apperrors.NewBadRequest(fmt.Sprintf("You are outside the allowed attendance area (%.0f m from %s, allowed radius %d m)",
		nearestDist, nearest.Name, nearest.RadiusMeters))
}

func validCoordinates(lat, lon *float64) bool {
	if lat != nil && (*lat < -90 || *lat > 90) {
		return false
	}
	if lon != nil && (*lon < -180 || *lon > 180) {
		return false
	}
	return true
}

func (uc *hrmUseCase) RecordClockIn(ctx context.Context, dto ClockInDTO) (*AttendanceResponseDTO, error) {
	dto.NIP = strings.TrimSpace(dto.NIP)
	dto.EmployeeName = strings.TrimSpace(dto.EmployeeName)
	if dto.NIP == "" || dto.EmployeeName == "" {
		return nil, apperrors.NewBadRequest("NIP and employee name are required")
	}
	if !validCoordinates(dto.Latitude, dto.Longitude) {
		return nil, apperrors.NewBadRequest("Invalid latitude/longitude")
	}

	now := uc.now().In(attendanceZone)
	dateStr := now.Format("2006-01-02")

	existing, err := uc.repo.FindAttendanceByNIPAndDate(ctx, dto.NIP, dateStr)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to check existing attendance")
	}
	if existing != nil {
		return nil, apperrors.NewConflict("Already clocked in today")
	}

	matched, err := uc.matchLocation(ctx, dto.Latitude, dto.Longitude)
	if err != nil {
		return nil, err
	}
	location := strings.TrimSpace(dto.Location)
	var locationID *uuid.UUID
	if matched != nil {
		location, locationID = matched.Name, &matched.ID
	} else if location == "" {
		location = "Unspecified"
	}
	method := dto.Method
	if method == "" {
		method = "GPS Mobile"
	}
	status := "On Time"
	if now.Hour() >= workStartHour {
		status = "Late"
	}

	att := &domain.Attendance{
		EmployeeName: dto.EmployeeName,
		NIP:          dto.NIP,
		Date:         dateStr,
		ClockIn:      now.Format("15:04 WIB"),
		ClockOut:     "--:--",
		ClockInAt:    &now,
		Location:     location,
		Method:       method,
		Status:       status,
		Latitude:     dto.Latitude,
		Longitude:    dto.Longitude,
		LocationID:   locationID,
		PhotoRef:     strings.TrimSpace(dto.PhotoRef),
	}
	if err := uc.repo.RecordAttendance(ctx, att); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to record clock in")
	}
	return ToAttendanceResponse(att), nil
}

func (uc *hrmUseCase) RecordClockOut(ctx context.Context, dto ClockOutDTO) (*AttendanceResponseDTO, error) {
	dto.NIP = strings.TrimSpace(dto.NIP)
	if dto.NIP == "" {
		return nil, apperrors.NewBadRequest("NIP is required")
	}
	if !validCoordinates(dto.Latitude, dto.Longitude) {
		return nil, apperrors.NewBadRequest("Invalid latitude/longitude")
	}

	now := uc.now().In(attendanceZone)
	att, err := uc.repo.FindAttendanceByNIPAndDate(ctx, dto.NIP, now.Format("2006-01-02"))
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to find today's attendance")
	}
	if att == nil {
		return nil, apperrors.NewNotFound("No clock-in found for today")
	}
	if att.ClockOutAt != nil || (att.ClockOut != "" && att.ClockOut != "--:--") {
		return nil, apperrors.NewConflict("Already clocked out today")
	}

	att.ClockOut = now.Format("15:04 WIB")
	att.ClockOutAt = &now
	if att.ClockInAt != nil && now.After(*att.ClockInAt) {
		att.WorkMinutes = int(now.Sub(*att.ClockInAt).Minutes())
	}
	if err := uc.repo.UpdateAttendance(ctx, att); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to record clock out")
	}
	return ToAttendanceResponse(att), nil
}

func toLocationResponse(l *domain.AttendanceLocation) AttendanceLocationResponseDTO {
	return AttendanceLocationResponseDTO{ID: l.ID, Name: l.Name, Latitude: l.Latitude, Longitude: l.Longitude, RadiusMeters: l.RadiusMeters, IsActive: l.IsActive}
}

func (uc *hrmUseCase) CreateAttendanceLocation(ctx context.Context, dto CreateAttendanceLocationDTO) (*AttendanceLocationResponseDTO, error) {
	name := strings.TrimSpace(dto.Name)
	if name == "" {
		return nil, apperrors.NewBadRequest("Location name is required")
	}
	if dto.Latitude < -90 || dto.Latitude > 90 || dto.Longitude < -180 || dto.Longitude > 180 {
		return nil, apperrors.NewBadRequest("Invalid latitude/longitude")
	}
	radius := dto.RadiusMeters
	if radius == 0 {
		radius = defaultGeofenceRadiusMeters
	}
	if radius < 0 {
		return nil, apperrors.NewBadRequest("Radius must be positive")
	}
	loc := &domain.AttendanceLocation{Name: name, Latitude: dto.Latitude, Longitude: dto.Longitude, RadiusMeters: radius, IsActive: true}
	if err := uc.repo.CreateAttendanceLocation(ctx, loc); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create attendance location")
	}
	res := toLocationResponse(loc)
	return &res, nil
}

func (uc *hrmUseCase) ListAttendanceLocations(ctx context.Context) ([]AttendanceLocationResponseDTO, error) {
	locs, err := uc.repo.ListAttendanceLocations(ctx, false)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list attendance locations")
	}
	out := make([]AttendanceLocationResponseDTO, len(locs))
	for i := range locs {
		out[i] = toLocationResponse(&locs[i])
	}
	return out, nil
}

func (uc *hrmUseCase) DeleteAttendanceLocation(ctx context.Context, id uuid.UUID) error {
	ok, err := uc.repo.DeleteAttendanceLocation(ctx, id)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to delete attendance location")
	}
	if !ok {
		return apperrors.NewNotFound("Attendance location not found")
	}
	return nil
}
