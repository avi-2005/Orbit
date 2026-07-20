package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// This is a deliberately simplified propagator — real two-body orbital
// mechanics plus J2 (Earth oblateness) secular drift on RAAN and argument
// of perigee — not a full SGP4 port (which also models atmospheric drag
// decay and several smaller perturbation terms). For visualizing LEO
// satellites over a single browser session this is accurate to within a
// small fraction of a degree, which is what actually matters here; it
// will drift further from a real SGP4 result over days, not minutes.

const (
	muEarth  = 398600.4418 // km^3/s^2, Earth's standard gravitational parameter
	earthReq = 6378.137    // km, equatorial radius
	j2       = 1.08263e-3  // Earth's oblateness coefficient
)

type OrbitalElements struct {
	Name          string
	Epoch         time.Time
	Inclination   float64 // radians
	RAAN0         float64 // radians
	Eccentricity  float64
	ArgPerigee0   float64 // radians
	MeanAnomaly0  float64 // radians
	MeanMotion    float64 // radians/second
	SemiMajorAxis float64 // km
}

// ParseTLE extracts orbital elements from a standard two-line element set.
// Column offsets follow the fixed-width TLE spec used since the 1980s.
func ParseTLE(name, line1, line2 string) (*OrbitalElements, error) {
	if len(line1) < 32 || len(line2) < 63 {
		return nil, fmt.Errorf("TLE lines too short for %s", name)
	}

	epochYear, err := strconv.Atoi(strings.TrimSpace(line1[18:20]))
	if err != nil {
		return nil, fmt.Errorf("bad epoch year: %w", err)
	}
	epochDay, err := strconv.ParseFloat(strings.TrimSpace(line1[20:32]), 64)
	if err != nil {
		return nil, fmt.Errorf("bad epoch day: %w", err)
	}
	year := 1900 + epochYear
	if epochYear < 57 {
		year = 2000 + epochYear
	}
	epoch := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC).
		Add(time.Duration((epochDay - 1) * 24 * float64(time.Hour)))

	inclDeg, err := strconv.ParseFloat(strings.TrimSpace(line2[8:16]), 64)
	if err != nil {
		return nil, fmt.Errorf("bad inclination: %w", err)
	}
	raanDeg, err := strconv.ParseFloat(strings.TrimSpace(line2[17:25]), 64)
	if err != nil {
		return nil, fmt.Errorf("bad RAAN: %w", err)
	}
	eccStr := strings.TrimSpace(line2[26:33])
	ecc, err := strconv.ParseFloat("0."+eccStr, 64)
	if err != nil {
		return nil, fmt.Errorf("bad eccentricity: %w", err)
	}
	argpDeg, err := strconv.ParseFloat(strings.TrimSpace(line2[34:42]), 64)
	if err != nil {
		return nil, fmt.Errorf("bad arg of perigee: %w", err)
	}
	maDeg, err := strconv.ParseFloat(strings.TrimSpace(line2[43:51]), 64)
	if err != nil {
		return nil, fmt.Errorf("bad mean anomaly: %w", err)
	}
	meanMotionRevDay, err := strconv.ParseFloat(strings.TrimSpace(line2[52:63]), 64)
	if err != nil {
		return nil, fmt.Errorf("bad mean motion: %w", err)
	}

	deg2rad := math.Pi / 180
	n := meanMotionRevDay * 2 * math.Pi / 86400 // rad/s
	a := math.Cbrt(muEarth / (n * n))           // semi-major axis, km

	return &OrbitalElements{
		Name:          strings.TrimSpace(name),
		Epoch:         epoch,
		Inclination:   inclDeg * deg2rad,
		RAAN0:         raanDeg * deg2rad,
		Eccentricity:  ecc,
		ArgPerigee0:   argpDeg * deg2rad,
		MeanAnomaly0:  maDeg * deg2rad,
		MeanMotion:    n,
		SemiMajorAxis: a,
	}, nil
}

// Propagate returns the satellite's current geodetic latitude, longitude,
// and altitude (km) at time t.
func Propagate(oe *OrbitalElements, t time.Time) (lat, lon, altKm float64) {
	dt := t.Sub(oe.Epoch).Seconds()

	p := oe.SemiMajorAxis * (1 - oe.Eccentricity*oe.Eccentricity)
	factor := 1.5 * oe.MeanMotion * j2 * (earthReq / p) * (earthReq / p)
	cosI := math.Cos(oe.Inclination)

	raanDot := -factor * cosI
	argpDot := 0.5 * factor * (5*cosI*cosI - 1)

	raan := oe.RAAN0 + raanDot*dt
	argp := oe.ArgPerigee0 + argpDot*dt
	meanAnomaly := math.Mod(oe.MeanAnomaly0+oe.MeanMotion*dt, 2*math.Pi)

	eccAnomaly := solveKepler(meanAnomaly, oe.Eccentricity)

	trueAnomaly := 2 * math.Atan2(
		math.Sqrt(1+oe.Eccentricity)*math.Sin(eccAnomaly/2),
		math.Sqrt(1-oe.Eccentricity)*math.Cos(eccAnomaly/2),
	)
	r := oe.SemiMajorAxis * (1 - oe.Eccentricity*math.Cos(eccAnomaly))

	xPf := r * math.Cos(trueAnomaly)
	yPf := r * math.Sin(trueAnomaly)

	cosRAAN, sinRAAN := math.Cos(raan), math.Sin(raan)
	cosArgp, sinArgp := math.Cos(argp), math.Sin(argp)
	cosI2, sinI2 := math.Cos(oe.Inclination), math.Sin(oe.Inclination)

	x := xPf*(cosRAAN*cosArgp-sinRAAN*sinArgp*cosI2) - yPf*(cosRAAN*sinArgp+sinRAAN*cosArgp*cosI2)
	y := xPf*(sinRAAN*cosArgp+cosRAAN*sinArgp*cosI2) - yPf*(sinRAAN*sinArgp-cosRAAN*cosArgp*cosI2)
	z := xPf*(sinArgp*sinI2) + yPf*(cosArgp*sinI2)

	gmst := greenwichMeanSiderealTime(t)
	ra := math.Atan2(y, x)
	lonRad := math.Mod(ra-gmst+3*math.Pi, 2*math.Pi) - math.Pi
	rxy := math.Sqrt(x*x + y*y)

	lat = math.Atan2(z, rxy) * 180 / math.Pi
	lon = lonRad * 180 / math.Pi
	altKm = math.Sqrt(x*x+y*y+z*z) - earthReq
	return lat, lon, altKm
}

func solveKepler(meanAnomaly, ecc float64) float64 {
	e := meanAnomaly
	for i := 0; i < 8; i++ {
		e = e - (e-ecc*math.Sin(e)-meanAnomaly)/(1-ecc*math.Cos(e))
	}
	return e
}

func julianDate(t time.Time) float64 {
	t = t.UTC()
	y, m, d := t.Year(), int(t.Month()), t.Day()
	if m <= 2 {
		y--
		m += 12
	}
	a := y / 100
	b := 2 - a + a/4
	dayFrac := float64(d) + (float64(t.Hour())+float64(t.Minute())/60+float64(t.Second())/3600)/24
	return math.Floor(365.25*float64(y+4716)) + math.Floor(30.6001*float64(m+1)) + dayFrac + float64(b) - 1524.5
}

func greenwichMeanSiderealTime(t time.Time) float64 {
	jd := julianDate(t)
	T := (jd - 2451545.0) / 36525.0
	gmstDeg := 280.46061837 + 360.98564736629*(jd-2451545.0) + 0.000387933*T*T - T*T*T/38710000.0
	gmstDeg = math.Mod(gmstDeg, 360)
	if gmstDeg < 0 {
		gmstDeg += 360
	}
	return gmstDeg * math.Pi / 180
}
