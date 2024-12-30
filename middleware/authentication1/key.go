package authentication

import (
	"verification-service/internal/library/middleware/keystore"
)

var key = keystore.Keys().Authentication()
