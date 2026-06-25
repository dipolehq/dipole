/*
geoip_condition.go evaluates geographic access restrictions.

Looks up the source IP in a MaxMind GeoLite2 database, extracts country and
subdivision codes, and checks against the configured allowlist. Fails closed:
if the DB is not loaded or IP lookup fails, the condition fails (deny).
*/
package conditions

import (
	"fmt"
	"net"

	"github.com/oschwald/geoip2-golang"
)

type GeoIPConditionChecker struct {
	db *geoip2.Reader
}

type GeoIPConditionConfig struct {
	Countries    []string `json:"countries"`
	Subdivisions []string `json:"subdivisions"`
}

func NewGeoIPConditionChecker(dbPath string) *GeoIPConditionChecker {
	if dbPath == "" {
		return &GeoIPConditionChecker{}
	}
	db, err := geoip2.Open(dbPath)
	if err != nil {
		return &GeoIPConditionChecker{}
	}
	return &GeoIPConditionChecker{db: db}
}

func (c *GeoIPConditionChecker) Check(sourceIP string, cfg *GeoIPConditionConfig) (bool, string, error) {
	if c.db == nil {
		return false, "GeoIP condition failed: database not loaded", nil
	}

	ip := net.ParseIP(sourceIP)
	if ip == nil {
		return false, fmt.Sprintf("GeoIP condition failed: invalid IP %q", sourceIP), nil
	}

	record, err := c.db.City(ip)
	if err != nil {
		return false, fmt.Sprintf("GeoIP condition failed: lookup error for %s", sourceIP), nil
	}

	country := record.Country.IsoCode
	subdivision := ""
	if len(record.Subdivisions) > 0 {
		subdivision = record.Subdivisions[0].IsoCode
	}

	if len(cfg.Countries) > 0 && !inList(country, cfg.Countries) {
		return false, fmt.Sprintf("GeoIP condition failed: source IP %s resolves to %s/%s, not in allowed countries %v",
			sourceIP, country, subdivision, cfg.Countries), nil
	}

	if len(cfg.Subdivisions) > 0 && !inList(subdivision, cfg.Subdivisions) {
		return false, fmt.Sprintf("GeoIP condition failed: source IP %s resolves to %s/%s, not in allowed subdivisions %v",
			sourceIP, country, subdivision, cfg.Subdivisions), nil
	}

	return true, "", nil
}

func inList(item string, list []string) bool {
	for _, v := range list {
		if v == item {
			return true
		}
	}
	return false
}
