module github.com/dipolehq/dipole/cloud

go 1.25.0

replace github.com/dipolehq/dipole/gen => ../gen

require (
	github.com/beevik/etree v1.5.0
	github.com/crewjam/saml v0.5.1
	github.com/dipolehq/dipole/gen v0.0.0-00010101000000-000000000000
	github.com/go-jose/go-jose/v4 v4.1.4
	github.com/golang-migrate/migrate/v4 v4.19.1
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.10.0
	github.com/oschwald/geoip2-golang v1.13.0
	github.com/russellhaering/goxmldsig v1.4.0
	go.mozilla.org/pkcs7 v0.10.0
	google.golang.org/grpc v1.81.1
)

require (
	github.com/jackc/pgerrcode v0.0.0-20220416144525-469b46aa5efa // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jonboulle/clockwork v0.2.2 // indirect
	github.com/mattermost/xml-roundtrip-validator v0.1.0 // indirect
	github.com/oschwald/maxminddb-golang v1.13.0 // indirect
	golang.org/x/crypto v0.48.0 // indirect
	golang.org/x/net v0.51.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260226221140-a57be14db171 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)
