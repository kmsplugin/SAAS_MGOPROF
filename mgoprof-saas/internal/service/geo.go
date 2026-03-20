package service

import (
	"net"

	"github.com/oschwald/maxminddb-golang"
	"go.uber.org/zap"

	"mgoprof-saas/internal/model"
)

type geoRecord struct {
	Country struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"country"`
	City struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"city"`
	Subdivisions []struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"subdivisions"`
}

type asnRecord struct {
	AutonomousSystemNumber       uint   `maxminddb:"autonomous_system_number"`
	AutonomousSystemOrganization string `maxminddb:"autonomous_system_organization"`
}

// GeoResolver resolves IP addresses to geographic locations and ASN/ISP info.
type GeoResolver struct {
	db     *maxminddb.Reader // GeoLite2-City or GeoIP2-City
	asnDB  *maxminddb.Reader // GeoLite2-ASN (optional)
	logger *zap.Logger
}

// NewGeoResolver opens GeoLite2-City (cityPath) and optionally GeoLite2-ASN (asnPath).
// Either path may be empty — the resolver degrades gracefully.
func NewGeoResolver(cityPath, asnPath string, logger *zap.Logger) *GeoResolver {
	r := &GeoResolver{logger: logger}

	if cityPath != "" {
		db, err := maxminddb.Open(cityPath)
		if err != nil {
			logger.Warn("geo city db open failed", zap.String("path", cityPath), zap.Error(err))
		} else {
			r.db = db
		}
	}

	if asnPath != "" {
		db, err := maxminddb.Open(asnPath)
		if err != nil {
			logger.Warn("geo asn db open failed", zap.String("path", asnPath), zap.Error(err))
		} else {
			r.asnDB = db
		}
	}

	return r
}

// Resolve returns Geo info (city + optionally ASN/ISP) for an IP string.
func (g *GeoResolver) Resolve(ip string) model.Geo {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return model.Geo{}
	}

	geo := model.Geo{}

	if g.db != nil {
		var rec geoRecord
		if err := g.db.Lookup(parsed, &rec); err == nil {
			geo.Country = rec.Country.Names["ru"]
			geo.City = rec.City.Names["ru"]
			if len(rec.Subdivisions) > 0 {
				geo.Region = rec.Subdivisions[0].Names["ru"]
			}
		}
	}

	if g.asnDB != nil {
		var rec asnRecord
		if err := g.asnDB.Lookup(parsed, &rec); err == nil {
			geo.ISPName = rec.AutonomousSystemOrganization
			if rec.AutonomousSystemNumber > 0 {
				// Format as "AS12345"
				geo.ISPASN = formatASN(rec.AutonomousSystemNumber)
			}
		}
	}

	return geo
}

func (g *GeoResolver) Close() {
	if g.db != nil {
		_ = g.db.Close()
	}
	if g.asnDB != nil {
		_ = g.asnDB.Close()
	}
}

func formatASN(n uint) string {
	// Quick itoa without fmt.Sprintf dependency
	if n == 0 {
		return "AS0"
	}
	digits := make([]byte, 0, 10)
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return "AS" + string(digits)
}
