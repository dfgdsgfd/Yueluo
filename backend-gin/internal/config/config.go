package config

import (
	"time"
)

type Config struct {
	Server           ServerConfig
	Log              LogConfig
	CORS             CORSConfig
	Database         DatabaseConfig
	Redis            RedisConfig
	Auth             AuthConfig
	API              APIConfig
	Debug            DebugConfig
	PyVideo          PyVideoConfig
	Frontend         FrontendConfig
	Balance          BalanceCenterConfig
	Creator          CreatorCenterConfig
	Upload           UploadConfig
	WebP             WebPConfig
	Video            VideoTranscodingConfig
	Audit            ContentAuditConfig
	Email            EmailConfig
	OAuth2           OAuth2Config
	Geetest          GeetestConfig
	IM               IMConfig
	Chat             ChatWSConfig
	Queue            QueueConfig
	Notify           NotificationChannelsConfig
	Media            MediaConfig
	Verify           VerificationConfig
	Observe          ObservabilityConfig
	AccessLog        AccessLogConfig
	SecurityAuditLog SecurityAuditLogConfig
	AccessBlock      AccessBlockConfig
	GeoIP            GeoIPConfig
}

type ServerConfig struct {
	Port              string
	Env               string
	ClientIPHeaders   []string
	RequestLogLevel   string
	RequestLogSilence []string
}

type LogConfig struct {
	Level             string
	FileEnabled       bool
	FileDir           string
	FileRetentionDays int
}

type GeoIPConfig struct {
	CountryMMDBPath string
	CountryMMDBURL  string
	SHA256URL       string
}

type AccessBlockConfig struct {
	Disabled                  bool
	VersionPollInterval       time.Duration
	ImportDisabled            bool
	ImportMinInterval         time.Duration
	ImportHTTPTimeout         time.Duration
	ImportMaxBytes            int64
	ImportSchedulerInterval   time.Duration
	ImportDefaultInterval     time.Duration
}

type CORSConfig struct {
	Origins []string
}

type DatabaseConfig struct {
	URL                    string
	Driver                 string
	Host                   string
	User                   string
	Password               string
	Name                   string
	Port                   int
	PoolSize               int
	MaxIdleConns           int
	ConnMaxLifetimeMS      int
	ConnectionTimeoutMS    int
	IdleTimeoutMS          int
	Charset                string
	Timezone               string
	Schema                 string
	SSLMode                string
	StatementTimeoutMS     int
	LockTimeoutMS          int
	LogLevel               string
	IgnoreRecordNotFound   bool
	PrepareStmt            bool
	CreateBatchSize        int
	SkipDefaultTransaction bool
	AutoMigrate            bool
}

type RedisConfig struct {
	Addr                string
	Password            string
	DB                  int
	PoolSize            int
	MinIdleConns        int
	CacheEnabled        bool
	CacheCommandTimeout time.Duration
	CacheDefaultTTL     time.Duration
	CacheL1TTL          time.Duration
	CacheL1MaxEntries   int
}

type AuthConfig struct {
	JWTSecret             string
	JWTExpiresIn          string
	AdminJWTExpiresIn     string
	RefreshTokenExpiresIn string
}

type DebugConfig struct {
	SwaggerDocsPath  string
	JWTTestTokenPath string
}

type PyVideoConfig struct {
	APIKey            string
	VideoCenterAPIKey string
	UpstreamURL       string
}

type APIConfig struct {
	BaseURL string
	Timeout time.Duration
}

type EmailConfig struct {
	Enabled  bool
	SMTP     SMTPConfig
	FromName string
	FromMail string
}

type SMTPConfig struct {
	Host     string
	Port     int
	Secure   bool
	Username string
	Password string
}

type OAuth2Config struct {
	Enabled         bool
	OnlyOAuth2      bool
	LoginURL        string
	ClientID        string
	ClientSecret    string
	RedirectURI     string
	RedirectBaseURL string
	CallbackPath    string
	AppCallbackURL  string
	EnableDPoP      bool
}

type GeetestConfig struct {
	Enabled    bool
	CaptchaID  string
	CaptchaKey string
	APIServer  string
}

type IMConfig struct {
	BaseURL    string
	HMACSecret string
	MessageKey string
	Mode       string
}

type FrontendConfig struct {
	BaseURL string
}

type BalanceCenterConfig struct {
	APIURL          string
	APIKey          string
	Timeout         time.Duration
	ExchangeRateIn  float64
	ExchangeRateOut float64
}

type CreatorCenterConfig struct {
	PlatformFeeRate         float64
	WithdrawEnabled         bool
	MinWithdrawAmount       float64
	ExtendedEarningsEnabled bool
	EarningsRates           CreatorEarningsRates
	DailyExtendedCap        float64
}

type CreatorEarningsRates struct {
	PerView     float64
	PerLike     float64
	PerCollect  float64
	PerComment  float64
	PerFollower float64
}

type UploadConfig struct {
	RootDir                   string
	LocalBase                 string
	MaxSize                   int64
	DeletePostUploadsOnDelete bool
	AvatarDir                 string
	BannerDir                 string
	Temp                      UploadTempConfig
	Recycle                   UploadRecycleConfig
	FileSigning               UploadFileSigningConfig
	Image                     UploadImageConfig
	Video                     UploadVideoConfig
	Attachment                UploadAttachmentConfig
}

type UploadRecycleConfig struct {
	Enabled         bool
	RootDir         string
	Retention       time.Duration
	CleanupInterval time.Duration
}

type UploadTempConfig struct {
	RootDir                   string
	CleanupInterval           time.Duration
	Retention                 time.Duration
	ProtectedPackageRetention time.Duration
}

type UploadFileSigningConfig struct {
	Secret string
	TTL    time.Duration
}

type UploadImageConfig struct {
	Strategy                string
	LocalUploadDir          string
	ImageHostAPIURL         string
	ImageHostTimeoutSeconds int
	MaxSizeBytes            int64
	R2                      R2Config
}

type UploadVideoConfig struct {
	Strategy        string
	LocalUploadDir  string
	LegacyUploadDir string
	CoverDir        string
	MaxSizeBytes    int64
	Chunk           UploadChunkConfig
	R2              R2Config
}

type UploadChunkConfig struct {
	TempDir           string
	ChunkSize         int64
	CleanupIntervalMS int64
	ExpireTimeMS      int64
}

type UploadAttachmentConfig struct {
	Strategy       string
	LocalUploadDir string
	MaxSizeBytes   int64
	R2             R2Config
}

type R2Config struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
	Endpoint        string
	PublicURL       string
	Region          string
}

type QueueConfig struct {
	Enabled     bool
	Concurrency QueueConcurrencyConfig
	Retry       QueueRetryConfig
}

type QueueConcurrencyConfig struct {
	IPLocation       int
	ContentAudit     int
	GeneralTask      int
	AITask           int
	VideoTranscoding int
	BatchNoteCreate  int
	ImageProtection  int
	AuditLog         int
}

type QueueRetryConfig struct {
	Attempts     int
	BackoffDelay time.Duration
}

type NotificationChannelsConfig struct {
	EmailEnabled       bool
	SystemEmailEnabled bool
	Discord            DiscordConfig
}

type DiscordConfig struct {
	Enabled    bool
	WebhookURL string
	SiteName   string
	SiteURL    string
}

type WebPConfig struct {
	EnableConversion  bool
	Quality           int
	AvatarQuality     int
	Method            int
	ConvertJPEG       bool
	ConvertPNG        bool
	KeepOriginal      bool
	MaxWidth          *int
	MaxHeight         *int
	Lossless          bool
	AlphaQuality      int
	HiddenWatermark   HiddenWatermarkConfig
	Watermark         WatermarkConfig
	UsernameWatermark WatermarkConfig
}

type HiddenWatermarkConfig struct {
	Secret string
	Remote HiddenWatermarkRemoteConfig
}

type HiddenWatermarkRemoteConfig struct {
	URL           string
	APIKey        string
	SkipTLSVerify bool
}

type WatermarkConfig struct {
	Enabled      bool
	Type         string
	Text         string
	FontSize     int
	FontPath     string
	ImagePath    string
	Opacity      int
	Position     string
	PositionMode string
	PreciseX     int
	PreciseY     int
	ImageRatio   int
	TileMode     bool
	Color        string
}

type VideoTranscodingConfig struct {
	Enabled            bool
	FFmpegPath         string
	FFprobePath        string
	MaxThreads         int
	MaxConcurrentTasks int
	OutputFormat       string
	DASH               DASHConfig
	DeleteOriginal     bool
	FFmpeg             FFmpegConfig
}

type DASHConfig struct {
	SegmentDuration    int
	MinBitrate         int
	MaxBitrate         int
	OriginalMaxBitrate int
	Resolutions        []DASHResolution
}

type DASHResolution struct {
	Width   int
	Height  int
	Bitrate int
	Label   string
}

type FFmpegConfig struct {
	Preset            string
	Profile           string
	CRF               *int
	GOPSize           *int
	BFrames           *int
	RefFrames         *int
	Complexity        *int
	AudioBitrate      int
	AudioSampleRate   int
	PixelFormat       string
	HardwareAccel     bool
	HardwareAccelType string
}

type ContentAuditConfig struct {
	Enabled bool
	APIURL  string
	APIKey  string
}

type ChatWSConfig struct {
	Enabled          bool
	BaseURL          string
	ProvisionSecret  string
	EncryptionKey    string
	TicketTTL        time.Duration
	HTTPTimeout      time.Duration
	MessageMaxLength int
}

type MediaConfig struct {
	AllowedDomains      []string
	RecommendationDebug bool
}

type VerificationConfig struct {
	CollectSensitiveInfo bool
}

type AccessLogConfig struct {
	Enabled       bool
	Scope         string
	Behaviors     []string
	Retention     time.Duration
	BufferSize    int
	BatchSize     int
	FlushInterval time.Duration
}

type SecurityAuditLogConfig struct {
	Enabled   bool
	Retention time.Duration
}

type ObservabilityConfig struct {
	SystemLogEnabled      bool
	SystemLogRetention    time.Duration
	MetricsEnabled        bool
	MetricsRetention      time.Duration
	MetricsBucket         time.Duration
	RuntimeSampleInterval time.Duration
	SlowRequestThreshold  time.Duration
	SlowQueryThreshold    time.Duration
}
