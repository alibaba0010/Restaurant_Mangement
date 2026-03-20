package providers

import (
	"github.com/alibaba0010/postgres-api/internal/common/types"
	"github.com/alibaba0010/postgres-api/internal/config"
)

// InitProviders initializes all supported payment providers
func InitProviders(cfg config.Config) map[types.PaymentProvider]PaymentProvider {
	providers := make(map[types.PaymentProvider]PaymentProvider)

	// Paystack
	if cfg.PAYSTACK_SECRET_KEY != "" {
		providers[types.PaymentProviderPaystack] = NewPaystackProvider(cfg.PAYSTACK_SECRET_KEY)
	}

	// Monnify
	if cfg.MONNIFY_API_KEY != "" && cfg.MONNIFY_SECRET_KEY != "" && cfg.MONNIFY_CONTRACT_CODE != "" {
		isProd := cfg.APP_ENV == "production"
		providers[types.PaymentProviderMonnify] = NewMonnifyProvider(cfg.MONNIFY_API_KEY, cfg.MONNIFY_SECRET_KEY, cfg.MONNIFY_CONTRACT_CODE, isProd)
	}

	// Flutterwave
	if cfg.FLUTTERWAVE_CLIENT_ID != "" && cfg.FLUTTERWAVE_CLIENT_SECRET != "" {
		providers[types.PaymentProviderFlutterwave] = NewFlutterwaveProvider(cfg.FLUTTERWAVE_CLIENT_ID, cfg.FLUTTERWAVE_CLIENT_SECRET, cfg.FLUTTERWAVE_ENCRYPTION_KEY, cfg.FLUTTERWAVE_BASE_URL, cfg.FLUTTERWAVE_AUTH_URL)
	}


	return providers
}
