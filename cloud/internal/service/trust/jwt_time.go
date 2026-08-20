// jwt_time.go provides shared time-based claim validation (exp, nbf) used by every JWT-based validator.
package trust

import "time"

const clockSkew = 180 * time.Second

func nbfNotYetValid(claims map[string]interface{}) bool {
	nbf, ok := claims["nbf"].(float64)
	if !ok {
		return false
	}
	return time.Unix(int64(nbf), 0).After(time.Now().Add(clockSkew))
}

func expIsExpired(claims map[string]interface{}) bool {
	exp, ok := claims["exp"].(float64)
	if !ok {
		return false
	}
	return int64(exp) < time.Now().Unix()
}
