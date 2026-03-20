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

// GeoResolver resolves IP addresses to geographic locations.
type GeoResolver struct {
	db     *maxminddb.Reader
	logger *zap.Logger
}

func NewGeoResolver(path string, logger *zap.Logger) *GeoResolver {
	if path == "" {
		return &GeoResolver{logger: logger}
	}
	db, err := maxminddb.Open(path)
	if err != nil {
		logger.Warn("geo db open failed, geo resolution disabled",
			zap.String("path", path),
			zap.Error(err),
		)
		return &GeoResolver{logger: logger}
	}
	return &GeoResolver{db: db, logger: logger}
}

func (g *GeoResolver) Resolve(ip string) model.Geo {
	if g.db == nil {
		return model.Geo{}
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return model.Geo{}
	}
	var rec geoRecord
	if err := g.db.Lookup(parsed, &rec); err != nil {
		return model.Geo{}
	}
	geo := model.Geo{
		Country: rec.Country.Names["ru"],
		City:    rec.City.Names["ru"],
	}
	if len(rec.Subdivisions) > 0 {
		geo.Region = rec.Subdivisions[0].Names["ru"]
	}
	return geo
}

func (g *GeoResolver) Close() {
	if g.db != nil {
		_ = g.db.Close()
	}
}
