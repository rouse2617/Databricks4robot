package config

import "os"

type Config struct {
	Env             string
	Port            string
	McapGatewayPort string
	DeliveryPort    string

	// Storage backend: bigtable | postgres
	StorageBackend string

	// Bigtable
	BigtableProject  string
	BigtableInstance string

	// PostgreSQL
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	// GCS (optional, for future derived artifacts)
	GCSProject       string
	GCSDerivedBucket string

	// Pub/Sub
	PubSubProject      string
	TopicMcapFinalized string
	TopicAssetEvents   string

	// Auth (Phase 0 static token; Phase 0.5 → OIDC)
	GraceToken string
}

func Load() *Config {
	return &Config{
		Env:             getenv("ENV", "development"),
		Port:            getenv("PORT", "8080"),
		McapGatewayPort: getenv("MCAP_GATEWAY_PORT", "8081"),
		DeliveryPort:    getenv("DELIVERY_PORT", "8082"),
		StorageBackend:  getenv("STORAGE_BACKEND", "bigtable"),

		BigtableProject:  getenv("BIGTABLE_PROJECT", ""),
		BigtableInstance: getenv("BIGTABLE_INSTANCE", "poc-datainfra"),

		DBHost:     getenv("DB_HOST", "localhost"),
		DBPort:     getenv("DB_PORT", "5432"),
		DBUser:     getenv("DB_USER", "postgres"),
		DBPassword: getenv("DB_PASSWORD", "postgres"),
		DBName:     getenv("DB_NAME", "data4cyber"),

		GCSProject:       getenv("GCS_PROJECT", ""),
		GCSDerivedBucket: getenv("GCS_DERIVED_BUCKET", "grace-derived"),

		PubSubProject:      getenv("PUBSUB_PROJECT", ""),
		TopicMcapFinalized: getenv("TOPIC_MCAP_FINALIZED", "gcs.mcap.finalized.v1"),
		TopicAssetEvents:   getenv("TOPIC_ASSET_EVENTS", "grace-asset-events"),

		GraceToken: getenv("GRACE_TOKEN", "dev-token"),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
