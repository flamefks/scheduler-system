package domain

func IsValidHTTPMethod(method string) bool {
	switch method {
	case "GET", "POST", "PUT", "PATCH", "DELETE":
		return true
	default:
		return false
	}
}

func IsValidBackoff(v string) bool {
	switch v {
	case "fixed", "exponential":
		return true
	default:
		return false
	}
}
func IsValidSourceDestinationType(v string) bool {
	switch v {
	case "http", "grpc":
		return true
	default:
		return false
	}
}
